// keyboard.js — キーボードショートカット
// ヘッダのトグルアイコンで有効/無効を切替え、状態は localStorage に保存する。
//
// ナビゲーション系（Gmail風の連続キー "g" → x）
//   g → p : ページ一覧
//   g → r : グラフビュー
//   g → n : 新規作成
//   g → i : トップ（移動後に検索窓にフォーカス）
//   g → g : 現在ページの先頭にスクロール（Vim風）
// ページ操作系
//   e : 編集（ページ閲覧時のみ）
//   j : 下にスクロール（ビューポート高の約50%）
//   k : 上にスクロール（ビューポート高の約50%）
//   G : 現在ページの末尾にスクロール（Vim風）
//
// 入力欄（input/textarea/contenteditable）にフォーカスがあるときはナビ系を無効化する。
// 編集画面では誤爆を避けるため、ショートカットを完全に無効化する。

(function() {
  var STORAGE_KEY = 'ivvvy.shortcuts';

  function isEnabled() {
    try {
      return localStorage.getItem(STORAGE_KEY) !== 'off';
    } catch (e) {
      return true;
    }
  }

  function setEnabled(enabled) {
    try {
      localStorage.setItem(STORAGE_KEY, enabled ? 'on' : 'off');
    } catch (e) {}
    syncToggleUI();
  }

  // ヘッダのトグルアイコンに現在状態を反映する。
  function syncToggleUI() {
    var btn = document.getElementById('shortcuts-toggle');
    if (!btn) return;
    var enabled = isEnabled();
    btn.classList.toggle('disabled', !enabled);
    btn.setAttribute('aria-pressed', enabled ? 'true' : 'false');
    var status = document.getElementById('shortcuts-status');
    if (status) status.textContent = enabled ? 'ON' : 'OFF';
  }

  function inEditableField() {
    var el = document.activeElement;
    if (!el) return false;
    var tag = el.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
    if (el.isContentEditable) return true;
    return false;
  }

  // Gmail風の連続キー "g" のシーケンス開始状態を保持する。
  // g を押した直後は1.5秒間だけ次キー待ちになる。
  var gPending = false;
  var gTimer = null;

  function startGSequence() {
    gPending = true;
    if (gTimer) clearTimeout(gTimer);
    gTimer = setTimeout(function() {
      gPending = false;
      gTimer = null;
    }, 1500);
  }

  function endGSequence() {
    gPending = false;
    if (gTimer) {
      clearTimeout(gTimer);
      gTimer = null;
    }
  }

  function navTargetFor(key) {
    switch (key) {
      case 'p': return '/pages';
      case 'r': return '/graph';
      case 'n': return '/new';
    }
    return null;
  }

  function focusSearch() {
    var input = document.getElementById('search-input');
    if (!input) return false;
    input.focus();
    input.select();
    return true;
  }

  function gotoEdit() {
    var link = document.querySelector('a.btn-edit');
    if (!link) return false;
    window.location.href = link.getAttribute('href');
    return true;
  }

  // 編集・新規作成画面は <form id="page-form"> を持つので、その有無で判定する。
  function isEditorPage() {
    return !!document.getElementById('page-form');
  }

  function onKeyDown(e) {
    // 編集画面ではすべてのショートカットを無効化する。
    if (isEditorPage()) return;

    if (!isEnabled()) return;

    if (e.metaKey || e.ctrlKey || e.altKey) return;

    var key = e.key;

    if (inEditableField()) {
      endGSequence();
      return;
    }

    if (gPending) {
      endGSequence();
      var lower = key.toLowerCase();
      if (lower === 'g') {
        e.preventDefault();
        window.scrollTo({ top: 0, behavior: 'smooth' });
        return;
      }
      // g i : トップへ遷移し、移動後に検索窓へフォーカスする。
      // 既に '/' にいる場合は遷移せず直接フォーカスを当てる。
      if (lower === 'i') {
        e.preventDefault();
        if (window.location.pathname === '/') {
          focusSearch();
        } else {
          window.location.href = '/?focus=search';
        }
        return;
      }
      var url = navTargetFor(lower);
      if (url) {
        e.preventDefault();
        window.location.href = url;
      }
      return;
    }

    if (key === 'g') {
      e.preventDefault();
      startGSequence();
      return;
    }

    if (key === 'G') {
      e.preventDefault();
      window.scrollTo({
        top: document.documentElement.scrollHeight,
        behavior: 'smooth'
      });
      return;
    }

    if (key === 'e') {
      if (gotoEdit()) {
        e.preventDefault();
      }
      return;
    }

    if (key === 'j') {
      e.preventDefault();
      window.scrollBy({ top: Math.round(window.innerHeight * 0.5), behavior: 'smooth' });
      return;
    }
    if (key === 'k') {
      e.preventDefault();
      window.scrollBy({ top: -Math.round(window.innerHeight * 0.5), behavior: 'smooth' });
      return;
    }
  }

  function onToggleClick(e) {
    e.preventDefault();
    setEnabled(!isEnabled());
  }

  function init() {
    var btn = document.getElementById('shortcuts-toggle');
    if (btn) {
      btn.addEventListener('click', onToggleClick);
      syncToggleUI();
    }
    document.addEventListener('keydown', onKeyDown);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
