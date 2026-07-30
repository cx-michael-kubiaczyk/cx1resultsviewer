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

  const highlightExtension = highlightRanges.length
    ? EditorView.decorations.of(
        Decoration.set(
          highlightRanges.map((h) =>
            Decoration.mark({
              class: closeFlowIds.has(h.flowId) ? "cm-node-highlight cm-node-badge" : "cm-node-highlight",
              attributes: { "data-flow-id": h.flowId, "data-node-number": String(h.displayNumber) },
            }).range(h.from, h.to)
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
