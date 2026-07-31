const SVG_NS = "http://www.w3.org/2000/svg";

export const MAX_CLOSE_LINE_GAP = 1;

export function linesAreClose(doc, posA, posB) {
  const lineA = doc.lineAt(posA).number;
  const lineB = doc.lineAt(posB).number;
  return Math.abs(lineA - lineB) <= MAX_CLOSE_LINE_GAP;
}

function areClose(a, b) {
  if (a.view !== b.view) return false;
  return linesAreClose(a.view.state.doc, a.to, b.from);
}

function buildEdges(boxes) {
  const byResult = new Map();

  for (const box of boxes) {
    for (const h of box.highlights) {
      const entry = {
        flowId: h.flowId,
        nodeIndex: h.nodeIndex,
        from: h.from,
        to: h.to,
        view: box.view,
        boxEl: box.containerEl,
      };
      if (!byResult.has(box.resultIndex)) byResult.set(box.resultIndex, []);
      byResult.get(box.resultIndex).push(entry);
    }
  }

  const edges = [];
  for (const [resultIndex, entries] of byResult) {
    entries.sort((a, b) => a.nodeIndex - b.nodeIndex);
    for (let i = 0; i < entries.length - 1; i++) {
      if (areClose(entries[i], entries[i + 1])) continue;
      edges.push({
        id: `edge-${resultIndex}-${entries[i].nodeIndex}-${entries[i + 1].nodeIndex}`,
        from: entries[i],
        to: entries[i + 1],
      });
    }
  }
  return edges;
}

function resolveEndpoint(entry, pos, preferRight) {
  const rect = entry.view.scrollDOM.getBoundingClientRect();
  const view = entry.view;

  let coords = null;
  if (pos >= view.viewport.from && pos <= view.viewport.to) {
    coords = view.coordsAtPos(pos);
  }

  if (coords && coords.top >= rect.top && coords.bottom <= rect.bottom) {
    return {
      x: preferRight ? coords.right : coords.left,
      y: (coords.top + coords.bottom) / 2,
      snapped: false,
    };
  }

  const before = pos < view.viewport.from || (coords && coords.top < rect.top);
  return {
    x: coords ? (coords.right + coords.left) / 2 : (rect.left + rect.right) / 2,
    y: before ? rect.top + 6 : rect.bottom - 6,
    snapped: true,
  };
}

function pathFor(p1, p2) {
  const dy = p2.y - p1.y;
  const offset = Math.min(Math.max(Math.abs(dy) * 0.3, 20), 80);
  const c1x = p1.x + offset;
  const c2x = p2.x - offset;
  return `M ${p1.x} ${p1.y} C ${c1x} ${p1.y}, ${c2x} ${p2.y}, ${p2.x} ${p2.y}`;
}

export function initFlowArrows(boxes) {
  const edges = buildEdges(boxes);
  if (!edges.length) return;

  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("class", "flow-arrows-overlay");

  const defs = document.createElementNS(SVG_NS, "defs");
  const marker = document.createElementNS(SVG_NS, "marker");
  marker.setAttribute("id", "flow-arrowhead");
  marker.setAttribute("viewBox", "0 0 10 10");
  marker.setAttribute("refX", "8");
  marker.setAttribute("refY", "5");
  marker.setAttribute("markerUnits", "userSpaceOnUse");
  marker.setAttribute("markerWidth", "12");
  marker.setAttribute("markerHeight", "12");
  marker.setAttribute("orient", "auto-start-reverse");
  const arrowPath = document.createElementNS(SVG_NS, "path");
  arrowPath.setAttribute("d", "M 0 0 L 10 5 L 0 10 z");
  arrowPath.setAttribute("class", "flow-arrowhead-fill");
  marker.appendChild(arrowPath);
  defs.appendChild(marker);
  svg.appendChild(defs);

  for (const edge of edges) {
    const path = document.createElementNS(SVG_NS, "path");
    path.setAttribute("class", "flow-arrow");
    path.setAttribute("marker-end", "url(#flow-arrowhead)");
    edge.pathEl = path;
    svg.appendChild(path);
  }

  document.body.appendChild(svg);

  function redrawAll() {
    svg.setAttribute("width", window.innerWidth);
    svg.setAttribute("height", window.innerHeight);

    for (const edge of edges) {
      const p1 = resolveEndpoint(edge.from, edge.from.to, true);
      const p2 = resolveEndpoint(edge.to, edge.to.from, false);
      const hidden = p1.snapped && p2.snapped;
      edge.pathEl.style.display = hidden ? "none" : "";
      if (hidden) continue;
      edge.pathEl.setAttribute("d", pathFor(p1, p2));
      edge.pathEl.classList.toggle("snapped", p1.snapped || p2.snapped);
    }
  }

  let rafId = null;
  function requestMeasure() {
    if (rafId) return;
    rafId = requestAnimationFrame(() => {
      rafId = null;
      redrawAll();
    });
  }

  window.addEventListener("scroll", requestMeasure, { passive: true });
  window.addEventListener("resize", requestMeasure);
  for (const box of boxes) {
    box.view.scrollDOM.addEventListener("scroll", requestMeasure, { passive: true });
  }

  const ro = new ResizeObserver(requestMeasure);
  for (const box of boxes) {
    ro.observe(box.containerEl);
    ro.observe(box.view.scrollDOM);
  }

  redrawAll();
}
