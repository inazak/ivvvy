// search.js — クライアントサイド全文検索
// サーバー側で生成されたインデックスJSONを取得し、
// ブラウザ上でAND方式の部分一致検索を実行する。
// 検索結果にはキーワード周辺の文字列（スニペット）も表示する。

(function() {
  // 検索インデックスのデータを保持する変数
  var searchIndex = null;

  // DOM要素の参照を取得する
  var input = document.getElementById('search-input');
  var results = document.getElementById('search-results');

  // 検索窓が存在しない場合は何もしない（検索窓がないページではスキップ）
  if (!input || !results) return;

  // サーバーから検索インデックスJSONを取得する。
  fetch('/search/index.json')
    .then(function(res) { return res.json(); })
    .then(function(data) {
      searchIndex = data;
    })
    .catch(function(err) {
      console.error('検索インデックスの読み込みに失敗:', err);
    });

  // 検索窓に入力があるたびに検索を実行する（300ms のデバウンス付き）
  var debounceTimer = null;
  input.addEventListener('input', function() {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function() {
      performSearch(input.value.trim());
    }, 300);
  });

  // blurイベントよりmousedownが先に発火するため、
  // このフラグでblur時の非表示を抑制できる。
  var isClickingResult = false;

  results.addEventListener('mousedown', function() {
    isClickingResult = true;
  });

  // 検索窓からフォーカスが外れたとき、少し遅延して結果を隠す。
  // ただし検索結果をクリック中の場合は非表示にしない。
  input.addEventListener('blur', function() {
    if (isClickingResult) {
      // 検索結果クリック中はblurで非表示にしない
      isClickingResult = false;
      return;
    }
    setTimeout(function() {
      results.classList.remove('active');
    }, 200);
  });

  // 検索窓にフォーカスしたとき、入力があれば結果を再表示する。
  input.addEventListener('focus', function() {
    if (input.value.trim() && results.children.length > 0) {
      results.classList.add('active');
    }
  });

  // performSearch は入力されたクエリでインデックスを検索し、結果を表示する。
  // クエリを空白で分割し、すべてのキーワードがマッチするエントリを返す。
  function performSearch(query) {
    // 結果表示をクリアする
    results.innerHTML = '';
    results.classList.remove('active');

    // クエリが空またはインデックス未読み込みの場合は何もしない
    if (!query || !searchIndex) return;

    // クエリを空白で分割してキーワード配列にする
    var keywords = query.toLowerCase().split(/\s+/);

    // 各インデックスエントリに対してキーワードマッチングを行う
    var matched = [];
    for (var i = 0; i < searchIndex.length; i++) {
      var entry = searchIndex[i];
      // タイトルと本文を結合した検索対象文字列を作る。
      var haystack = (entry.title + ' ' + entry.body).toLowerCase();

      // すべてのキーワードが含まれているかチェックする（AND検索）
      var allMatch = true;
      for (var k = 0; k < keywords.length; k++) {
        if (haystack.indexOf(keywords[k]) === -1) {
          allMatch = false;
          break;
        }
      }

      if (allMatch) {
        matched.push(entry);
      }
    }

    // マッチした結果がなければ何も表示しない
    if (matched.length === 0) return;

    // 検索結果をDOM要素として追加する（最大20件）
    var limit = Math.min(matched.length, 20);
    for (var j = 0; j < limit; j++) {
      var item = matched[j];

      // キーワード周辺のスニペットを生成する
      var snippet = buildSnippet(item.body || '', keywords);

      var div = document.createElement('div');
      div.className = 'search-result-item';
      div.innerHTML = '<div class="title">' + escapeHTML(item.title) + '</div>' +
                      (snippet ? '<div class="snippet">' + snippet + '</div>' : '');
      div.dataset.id = item.id;

      // クリックでページに遷移する
      div.addEventListener('click', function() {
        window.location.href = '/page/' + this.dataset.id;
      });

      results.appendChild(div);
    }

    results.classList.add('active');
  }

  // buildSnippet は本文中からキーワードが出現する箇所を見つけ、
  // 前後40文字程度のスニペットを生成する。
  // キーワード部分は <mark> タグでハイライトする。
  function buildSnippet(body, keywords) {
    if (!body) return '';

    var bodyLower = body.toLowerCase();
    var bestPos = -1;
    var bestKeyword = '';

    // 最初にマッチするキーワードの位置を探す
    for (var i = 0; i < keywords.length; i++) {
      var pos = bodyLower.indexOf(keywords[i]);
      if (pos >= 0 && (bestPos < 0 || pos < bestPos)) {
        bestPos = pos;
        bestKeyword = keywords[i];
      }
    }

    if (bestPos < 0) return '';

    // キーワードの前後40文字を切り出す
    var start = Math.max(0, bestPos - 40);
    var end = Math.min(body.length, bestPos + bestKeyword.length + 40);
    var fragment = body.substring(start, end);

    // 改行を空白に置換する
    fragment = fragment.replace(/\n/g, ' ');

    // 先頭・末尾に省略記号を付ける
    if (start > 0) fragment = '...' + fragment;
    if (end < body.length) fragment = fragment + '...';

    // キーワード部分をハイライトする
    var escaped = escapeHTML(fragment);
    for (var j = 0; j < keywords.length; j++) {
      var kw = keywords[j];
      // 大文字小文字を無視してハイライトする
      var regex = new RegExp('(' + escapeRegex(escapeHTML(kw)) + ')', 'gi');
      escaped = escaped.replace(regex, '<mark>$1</mark>');
    }

    return escaped;
  }

  // escapeHTML はXSS防止のためにHTML特殊文字をエスケープする。
  function escapeHTML(str) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }

  // escapeRegex は正規表現の特殊文字をエスケープする。
  function escapeRegex(str) {
    return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }
})();
