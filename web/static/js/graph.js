// graph.js — ページ間 WikiLink を可視化するグラフビュー。
// /api/graph.json から { nodes, edges } を取得し、cytoscape.js で描画する。
// ivy（ツタ）にちなんで配色は緑基調。ライト/ダーク両モードに対応する。

(function() {
  // テーマ別のスタイル定義。ライト＝深緑、ダーク＝明るい緑。
  // ノードの選択色とエッジの強調色は、ホバー時のアクセントに使う。
  var PALETTE = {
    light: {
      nodeBg:   '#4a8a5a',
      nodeBorder: '#2d5a3d',
      nodeLabel: '#2d3a2d',
      edge:     '#7ab088',
      accent:   '#6dbb83'
    },
    dark: {
      nodeBg:   '#6dbb83',
      nodeBorder: '#9dd4a8',
      nodeLabel: '#d8e8d8',
      edge:     '#4a7a58',
      accent:   '#a8e6b8'
    }
  };

  function currentPalette() {
    var theme = document.documentElement.getAttribute('data-theme');
    return theme === 'dark' ? PALETTE.dark : PALETTE.light;
  }

  function buildStyle(p) {
    return [
      {
        selector: 'node',
        style: {
          'background-color': p.nodeBg,
          'border-color': p.nodeBorder,
          'border-width': 2,
          'label': 'data(title)',
          'color': p.nodeLabel,
          'font-size': '10px',
          'text-valign': 'bottom',
          'text-margin-y': 4,
          'text-max-width': '120px',
          'text-wrap': 'ellipsis',
          'text-background-opacity': 0,
          'width': 18,
          'height': 18
        }
      },
      {
        selector: 'node:hover, node:selected',
        style: {
          'background-color': p.accent,
          'border-color': p.accent,
          'font-size': '12px',
          'z-index': 100
        }
      },
      {
        selector: 'edge',
        style: {
          'width': 1.5,
          'line-color': p.edge,
          'target-arrow-color': p.edge,
          'target-arrow-shape': 'triangle',
          'curve-style': 'bezier',
          'arrow-scale': 0.8
        }
      },
      {
        selector: 'edge:hover, edge.highlighted',
        style: {
          'line-color': p.accent,
          'target-arrow-color': p.accent,
          'width': 2.5
        }
      }
    ];
  }

  function initGraph(data) {
    var container = document.getElementById('cy');
    if (!container || !window.cytoscape) return;

    var elements = [];
    for (var i = 0; i < data.nodes.length; i++) {
      var n = data.nodes[i];
      elements.push({ group: 'nodes', data: { id: n.id, title: n.title || n.id } });
    }
    for (var j = 0; j < data.edges.length; j++) {
      var e = data.edges[j];
      elements.push({
        group: 'edges',
        data: { id: e.source + '->' + e.target, source: e.source, target: e.target }
      });
    }

    var cy = window.cytoscape({
      container: container,
      elements: elements,
      style: buildStyle(currentPalette()),
      layout: {
        // cose: Compound Spring Embedder（cytoscape 同梱、追加プラグイン不要）。
        // 孤立ノードも周辺に自然配置される。
        name: 'cose',
        animate: false,
        nodeRepulsion: 8000,
        idealEdgeLength: 80,
        gravity: 0.25,
        padding: 30
      },
      wheelSensitivity: 0.2,
      minZoom: 0.2,
      maxZoom: 3
    });

    // ノードクリックで該当ページへ遷移する。
    cy.on('tap', 'node', function(evt) {
      var id = evt.target.id();
      if (id) window.location.href = '/page/' + encodeURIComponent(id);
    });

    // テーマ切替（data-theme 属性変更）時にスタイルを再適用する。
    var observer = new MutationObserver(function(mutations) {
      for (var k = 0; k < mutations.length; k++) {
        if (mutations[k].attributeName === 'data-theme') {
          cy.style(buildStyle(currentPalette())).update();
          return;
        }
      }
    });
    observer.observe(document.documentElement, { attributes: true });
  }

  function load() {
    fetch('/api/graph.json', { headers: { 'Accept': 'application/json' } })
      .then(function(res) {
        if (!res.ok) throw new Error('graph.json fetch failed: ' + res.status);
        return res.json();
      })
      .then(function(data) {
        initGraph(data || { nodes: [], edges: [] });
      })
      .catch(function(err) {
        var el = document.getElementById('cy');
        if (el) {
          el.innerHTML = '<p style="padding:1rem;color:#a00">' + window.IVVVY_I18N.graphDataFailed + ': '
            + (err && err.message ? err.message : err) + '</p>';
        }
      });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', load);
  } else {
    load();
  }
})();
