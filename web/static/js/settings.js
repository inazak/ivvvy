// settings.js — テキストサイズとカラーテーマの設定管理
//
// テキストサイズ:
//   localStorage キー 'ivvvy.textsize' に 'large' / 'normal' / 'small' を保存する。
//   html 要素の data-textsize 属性を切り替え、CSS の :root font-size を変更する。
//
// カラーテーマ:
//   localStorage キー 'ivvvy.theme' に '' (ライト) または 'dark' を保存する。
//   html 要素の data-theme 属性を切り替える。
//   ヘッダのトグルアイコン（月/太陽）を1クリックすると2モードを切り替える。
//   旧テーマ名（nordic, cyber, luxury, chiffon）が保存されている場合は
//   ライトモードにフォールバックする。
//
// FOUC（Flash of Unstyled Content）防止のため、layout.html の <head> 内に
// インラインスクリプトで data-textsize / data-theme を即座に復元している。
// このファイルでは DOM ready 後の UI 初期化とイベント登録を担当する。

(function() {
  var SIZE_KEY = 'ivvvy.textsize';
  var THEME_KEY = 'ivvvy.theme';

  // === テキストサイズ管理 ===

  // localStorage が使えない環境ではデフォルト 'normal' を返す。
  function getTextSize() {
    try { return localStorage.getItem(SIZE_KEY) || 'normal'; }
    catch(e) { return 'normal'; }
  }

  // 'normal' の場合は属性自体を削除して、CSS のデフォルト値（16px）を使う。
  function applyTextSize(size) {
    if (size && size !== 'normal') {
      document.documentElement.setAttribute('data-textsize', size);
    } else {
      document.documentElement.removeAttribute('data-textsize');
    }
  }

  function setTextSize(size) {
    try { localStorage.setItem(SIZE_KEY, size); } catch(e) {}
    applyTextSize(size);
    syncTextSizeUI(size);
  }

  // 現在選択中のサイズに対応するボタンに 'active' クラスを付与する。
  function syncTextSizeUI(size) {
    var popup = document.querySelector('.textsize-popup');
    if (!popup) return;
    var buttons = popup.querySelectorAll('button[data-size]');
    for (var i = 0; i < buttons.length; i++) {
      var btn = buttons[i];
      if (btn.getAttribute('data-size') === size) {
        btn.classList.add('active');
      } else {
        btn.classList.remove('active');
      }
    }
  }

  // === カラーテーマ管理 ===

  // '' (空文字) はライトモード、'dark' はダークモードを意味する。
  var VALID_THEMES = { '': true, 'dark': true };

  // 未設定や廃止された旧テーマ名が入っている場合はライトモード（空文字）にフォールバックする。
  function getTheme() {
    try {
      var v = localStorage.getItem(THEME_KEY) || '';
      return VALID_THEMES[v] ? v : '';
    } catch(e) { return ''; }
  }

  // 空文字（ライトモード）の場合は属性自体を削除して、:root の CSS 変数をそのまま使う。
  function applyTheme(theme) {
    if (theme === 'dark') {
      document.documentElement.setAttribute('data-theme', 'dark');
    } else {
      document.documentElement.removeAttribute('data-theme');
    }
  }

  function setTheme(theme) {
    if (!VALID_THEMES[theme]) theme = '';
    try { localStorage.setItem(THEME_KEY, theme); } catch(e) {}
    applyTheme(theme);
  }

  function toggleTheme() {
    setTheme(getTheme() === 'dark' ? '' : 'dark');
  }

  // === ポップアップ開閉管理 ===

  function togglePopup(wrapEl) {
    var isOpen = wrapEl.classList.contains('open');
    closeAllPopups();
    if (!isOpen) {
      wrapEl.classList.add('open');
    }
  }

  function closeAllPopups() {
    var wraps = document.querySelectorAll('.textsize-wrap');
    for (var i = 0; i < wraps.length; i++) {
      wraps[i].classList.remove('open');
    }
  }

  // === DOM ready 後の初期化 ===

  function init() {
    var textsizeToggle = document.getElementById('textsize-toggle');
    var textsizeWrap = document.querySelector('.textsize-wrap');
    var textsizePopup = document.querySelector('.textsize-popup');

    if (textsizeToggle && textsizeWrap) {
      textsizeToggle.addEventListener('click', function(e) {
        e.preventDefault();
        e.stopPropagation();
        togglePopup(textsizeWrap);
      });
    }

    if (textsizePopup) {
      var sizeButtons = textsizePopup.querySelectorAll('button[data-size]');
      for (var i = 0; i < sizeButtons.length; i++) {
        sizeButtons[i].addEventListener('click', function(e) {
          e.preventDefault();
          e.stopPropagation();
          setTextSize(this.getAttribute('data-size'));
          closeAllPopups();
        });
      }
    }

    syncTextSizeUI(getTextSize());

    var themeToggle = document.getElementById('theme-toggle');

    if (themeToggle) {
      themeToggle.addEventListener('click', function(e) {
        e.preventDefault();
        e.stopPropagation();
        toggleTheme();
      });
    }

    // 旧テーマ名が localStorage に残っている場合に備えて applyTheme を再実行する。
    applyTheme(getTheme());

    document.addEventListener('click', function() {
      closeAllPopups();
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
