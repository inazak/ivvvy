// editor.js — Markdownプレビュー付きエディタ
// サーバーサイドAPIでMarkdownをHTMLに変換してプレビュー表示する。
// タブ切り替え方式で「編集」と「プレビュー」を全幅で表示する。

var selectedTags = [];
var allAvailableTags = [];

function initEditor(isNew) {
  var form = document.getElementById('page-form');
  var body = document.getElementById('page-body');
  var preview = document.getElementById('preview');
  var deleteBtn = document.getElementById('delete-btn');

  if (!form || !body) return;

  selectedTags = (typeof initialTags !== 'undefined' && initialTags) ? initialTags.slice() : [];
  initTagInput();

  var tabBtns = document.querySelectorAll('.tab-btn');
  for (var i = 0; i < tabBtns.length; i++) {
    tabBtns[i].addEventListener('click', function() {
      var targetTab = this.getAttribute('data-tab');

      for (var j = 0; j < tabBtns.length; j++) {
        tabBtns[j].classList.remove('active');
      }
      this.classList.add('active');

      var contents = document.querySelectorAll('.tab-content');
      for (var k = 0; k < contents.length; k++) {
        contents[k].classList.remove('active');
      }
      document.getElementById('tab-' + targetTab).classList.add('active');

      if (targetTab === 'preview') {
        updatePreview(body.value, preview);
      }
    });
  }

  form.addEventListener('submit', function(e) {
    e.preventDefault();
    savePage(isNew);
  });

  if (deleteBtn) {
    deleteBtn.addEventListener('click', function() {
      if (confirm(window.IVVVY_I18N.confirmDelete)) {
        deletePage();
      }
    });
  }
}

function initTagInput() {
  var container = document.getElementById('tag-input-container');
  var input = document.getElementById('tag-input');
  var suggestionsEl = document.getElementById('tag-suggestions');
  if (!container || !input || !suggestionsEl) return;

  fetch('/api/tags')
    .then(function(res) { return res.json(); })
    .then(function(tags) { allAvailableTags = tags || []; renderSuggestions(''); })
    .catch(function() { allAvailableTags = []; });

  renderTagPills();

  container.addEventListener('click', function() { input.focus(); });

  input.addEventListener('input', function() {
    renderSuggestions(input.value.trim());
  });

  input.addEventListener('keydown', function(e) {
    var val = input.value.trim();
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      if (val && selectedTags.indexOf(val) === -1) {
        selectedTags.push(val);
        input.value = '';
        renderTagPills();
        renderSuggestions('');
      }
    }
    if (e.key === 'Backspace' && !input.value && selectedTags.length > 0) {
      selectedTags.pop();
      renderTagPills();
      renderSuggestions('');
    }
  });
}

function renderTagPills() {
  var container = document.getElementById('tag-input-container');
  var input = document.getElementById('tag-input');
  var existing = container.querySelectorAll('.tag-pill');
  for (var i = 0; i < existing.length; i++) {
    existing[i].remove();
  }
  for (var j = 0; j < selectedTags.length; j++) {
    var pill = document.createElement('span');
    pill.className = 'tag-pill';
    pill.textContent = selectedTags[j];
    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'tag-pill-remove';
    btn.textContent = '×';
    btn.setAttribute('data-tag', selectedTags[j]);
    btn.addEventListener('click', function() {
      var tag = this.getAttribute('data-tag');
      selectedTags = selectedTags.filter(function(t) { return t !== tag; });
      renderTagPills();
      renderSuggestions(document.getElementById('tag-input').value.trim());
    });
    pill.appendChild(btn);
    container.insertBefore(pill, input);
  }
}

function renderSuggestions(filter) {
  var el = document.getElementById('tag-suggestions');
  if (!el) return;
  el.innerHTML = '';

  var candidates = allAvailableTags.filter(function(tag) {
    if (selectedTags.indexOf(tag) !== -1) return false;
    if (filter && tag.toLowerCase().indexOf(filter.toLowerCase()) === -1) return false;
    return true;
  });

  for (var i = 0; i < candidates.length; i++) {
    var btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'tag-suggestion';
    btn.textContent = candidates[i];
    btn.setAttribute('data-tag', candidates[i]);
    btn.addEventListener('click', function() {
      var tag = this.getAttribute('data-tag');
      if (selectedTags.indexOf(tag) === -1) {
        selectedTags.push(tag);
        var input = document.getElementById('tag-input');
        input.value = '';
        renderTagPills();
        renderSuggestions('');
        input.focus();
      }
    });
    el.appendChild(btn);
  }
}

// savePage はフォームのデータをAPIに送信してページを保存する。
// isNew が true なら POST（新規作成）、false なら PUT（更新）を使う。
// 新規作成時はIDを送らず、サーバー側でミリ秒タイムスタンプを自動採番する（#3）。
function savePage(isNew) {
  var idEl = document.getElementById('page-id');
  var id = idEl ? idEl.value : '';
  var title = document.getElementById('page-title').value;
  var body = document.getElementById('page-body').value;

  var data = {
    title: title,
    body: body,
    tags: selectedTags
  };

  // 新規作成か更新かでAPIエンドポイントとHTTPメソッドを分ける。
  // 新規作成時はIDをサーバーに任せる。
  var url, method;
  if (isNew) {
    url = '/api/page';
    method = 'POST';
  } else {
    url = '/api/page/' + id;
    method = 'PUT';
    data.id = id;
  }

  fetch(url, {
    method: method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  })
  .then(function(res) {
    if (!res.ok) {
      return res.text().then(function(t) { throw new Error(t); });
    }
    return res.json();
  })
  .then(function(result) {
    // 保存成功後、ページ閲覧画面に遷移する
    window.location.href = '/page/' + (result.id || id);
  })
  .catch(function(err) {
    alert(window.IVVVY_I18N.saveFailed + ': ' + err.message);
  });
}

// deletePage は現在のページを削除するAPIを呼び出す。
function deletePage() {
  var id = document.getElementById('page-id').value;

  fetch('/api/page/' + id, { method: 'DELETE' })
  .then(function(res) {
    if (!res.ok) {
      return res.text().then(function(t) { throw new Error(t); });
    }
    return res.json();
  })
  .then(function() {
    // 削除成功後、トップページに遷移する
    window.location.href = '/';
  })
  .catch(function(err) {
    alert(window.IVVVY_I18N.deleteFailed + ': ' + err.message);
  });
}

// updatePreview はサーバーサイドAPIにMarkdownを送信し、
// 変換されたHTMLをプレビューペインに表示する。
// これにより、GFMテーブル・全レベル見出し・WikiLink等が正確に反映される（#1）。
function updatePreview(markdown, previewEl) {
  if (!previewEl) return;
  if (!markdown) {
    previewEl.innerHTML = '<p style="color:#999">' + window.IVVVY_I18N.previewPlaceholder + '</p>';
    return;
  }

  // サーバーサイドのプレビューAPIを呼び出す
  fetch('/api/preview', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ body: markdown })
  })
  .then(function(res) { return res.json(); })
  .then(function(data) {
    previewEl.innerHTML = '<div class="page-content">' + data.html + '</div>';
  })
  .catch(function(err) {
    previewEl.innerHTML = '<p style="color:red">' + window.IVVVY_I18N.previewFailed + '</p>';
  });
}
