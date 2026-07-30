import { EditorView, basicSetup } from "codemirror";
import { EditorState, Text } from "@codemirror/state";
import { Decoration } from "@codemirror/view";
import { initFlowArrows, linesAreClose } from "./flowArrows.js";

function computeRange(doc, line, column, length) {
  const clampedLine = Math.min(Math.max(Number(line), 1), doc.lines);
  const lineInfo = doc.line(clampedLine);
  const from = Math.min(lineInfo.from + Math.max(Number(column) - 1, 0), lineInfo.to);
  const to = Math.min(from + Math.max(Number(length), 0), doc.length);
  return { from, to };
}

function applyDefaultLayout(view, doc, minLine, maxLine) {
  const lineHeight = view.defaultLineHeight;
  const displayLines = Math.min(Math.max(Number(maxLine) - Number(minLine) + 1, 1), 20);
  const height = Math.ceil(lineHeight * displayLines);
  view.dom.style.height = height + "px";
  view.requestMeasure();

  const midLine = Math.round((Number(minLine) + Number(maxLine)) / 2);
  const clampedMidLine = Math.min(Math.max(midLine, 1), doc.lines);
  const midPos = doc.line(clampedMidLine).from;
  view.dispatch({ effects: EditorView.scrollIntoView(midPos, { y: "center" }) });
}

function formatNodeLabel(displayNumbers) {
  const sorted = [...displayNumbers].sort((a, b) => a - b);
  const parts = [];
  let start = sorted[0];
  let prev = sorted[0];
  for (let i = 1; i <= sorted.length; i++) {
    const cur = sorted[i];
    if (cur !== prev + 1) {
      parts.push(start === prev ? `${start}` : `${start}-${prev}`);
      start = cur;
    }
    prev = cur;
  }
  return parts.join(",");
}

// Nodes that start at the exact same document position (same line/column,
// e.g. a variable read on every loop iteration) would otherwise render as
// stacked ::before badges at the same spot, hiding all but the first. Merge
// them into a single decoration spanning the largest of the overlapping
// nodes, labeled with the combined node range/list.
function buildDecorationRanges(highlightRanges, closeFlowIds) {
  const decorations = [];
  let i = 0;
  while (i < highlightRanges.length) {
    let j = i;
    while (j + 1 < highlightRanges.length && highlightRanges[j + 1].from === highlightRanges[i].from) {
      j++;
    }
    if (j === i) {
      const h = highlightRanges[i];
      decorations.push({
        from: h.from,
        to: h.to,
        flowId: h.flowId,
        label: String(h.displayNumber),
        showBadge: closeFlowIds.has(h.flowId),
      });
    } else {
      const group = highlightRanges.slice(i, j + 1);
      decorations.push({
        from: group[0].from,
        to: Math.max(...group.map((h) => h.to)),
        flowId: group.map((h) => h.flowId).join(","),
        label: formatNodeLabel(group.map((h) => h.displayNumber)),
        showBadge: true,
      });
    }
    i = j + 1;
  }
  return decorations;
}

function mountBox(container) {
  const payloadEl = container.querySelector(".codebox-payload");
  const data = JSON.parse(payloadEl.textContent);
  const mountEl = container.querySelector(".codebox-editor");
  const headerEl = container.querySelector(".codebox-header");
  const stepsEl = container.querySelector(".codebox-steps");

  const doc = Text.of(data.source.split("\n"));

  // data.highlights is already in flow (nodeIndex-ascending) order, which is
  // what we need to detect same-line/adjacent-line neighbours before it gets
  // re-sorted by document position below.
  const orderedHighlights = data.highlights
    .map((h, i) => {
      const { from, to } = computeRange(doc, h.line, h.column, h.length);
      return { flowId: h.flowId, nodeIndex: h.nodeIndex, displayNumber: i + 1, from, to };
    })
    .filter((h) => h.to > h.from);

  const closeFlowIds = new Set();
  for (let i = 0; i < orderedHighlights.length - 1; i++) {
    const a = orderedHighlights[i];
    const b = orderedHighlights[i + 1];
    if (linesAreClose(doc, a.to, b.from)) {
      closeFlowIds.add(a.flowId);
      closeFlowIds.add(b.flowId);
    }
  }

  const highlightRanges = orderedHighlights.sort((a, b) => a.from - b.from || a.to - b.to);

  const decorationRanges = buildDecorationRanges(highlightRanges, closeFlowIds);

  const highlightExtension = decorationRanges.length
    ? EditorView.decorations.of(
        Decoration.set(
          decorationRanges.map((d) =>
            Decoration.mark({
              class: d.showBadge ? "cm-node-highlight cm-node-badge" : "cm-node-highlight",
              attributes: { "data-flow-id": d.flowId, "data-node-number": d.label },
            }).range(d.from, d.to)
          )
        )
      )
    : [];

  const view = new EditorView({
    state: EditorState.create({
      doc,
      extensions: [
        basicSetup,
        EditorState.readOnly.of(true),
        EditorView.editable.of(false),
        highlightExtension,
      ],
    }),
    parent: mountEl,
  });

  requestAnimationFrame(() => {
    applyDefaultLayout(view, doc, data.minLine, data.maxLine);
  });

  return {
    boxId: data.boxId,
    resultIndex: data.resultIndex,
    view,
    doc,
    minLine: data.minLine,
    maxLine: data.maxLine,
    containerEl: container,
    headerEl,
    stepsEl,
    editorEl: mountEl,
    highlights: highlightRanges,
  };
}

function wireButtons(box) {
  box.containerEl.querySelector('[data-action="maximize"]').addEventListener("click", () => {
    const top = box.editorEl.getBoundingClientRect().top;
    box.view.dom.style.height = Math.max(window.innerHeight - top - 16, 40) + "px";
    box.view.requestMeasure();
  });

  box.containerEl.querySelector('[data-action="reset"]').addEventListener("click", () => {
    applyDefaultLayout(box.view, box.doc, box.minLine, box.maxLine);
  });

  const toggleBtn = box.containerEl.querySelector('[data-action="toggle-list"]');
  toggleBtn.addEventListener("click", () => {
    const hidden = box.stepsEl.classList.toggle("is-hidden");
    toggleBtn.setAttribute("aria-pressed", String(!hidden));
  });
}

function wireResizeHandles(box) {
  const MIN_HEIGHT = Math.max(box.view.defaultLineHeight * 1.5, 24);
  box.containerEl.querySelectorAll('[data-role="resize-handle"]').forEach((handle) => {
    const sign = handle.dataset.edge === "bottom" ? 1 : -1;

    handle.addEventListener("pointerdown", (e) => {
      e.preventDefault();
      handle.setPointerCapture(e.pointerId);
      handle.classList.add("is-dragging");
      const startY = e.clientY;
      const startHeight = box.view.dom.getBoundingClientRect().height;
      document.body.style.userSelect = "none";

      function onMove(ev) {
        const delta = (ev.clientY - startY) * sign;
        box.view.dom.style.height = Math.max(startHeight + delta, MIN_HEIGHT) + "px";
      }
      function onUp(ev) {
        handle.releasePointerCapture(ev.pointerId);
        handle.classList.remove("is-dragging");
        document.body.style.userSelect = "";
        box.view.requestMeasure();
        handle.removeEventListener("pointermove", onMove);
        handle.removeEventListener("pointerup", onUp);
      }
      handle.addEventListener("pointermove", onMove);
      handle.addEventListener("pointerup", onUp);
    });
  });
}

const boxes = Array.from(document.querySelectorAll('[data-role="codebox"]')).map(mountBox);
boxes.forEach(wireButtons);
boxes.forEach(wireResizeHandles);
initFlowArrows(boxes);
