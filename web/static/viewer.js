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

function mountBox(container) {
  const payloadEl = container.querySelector(".codebox-payload");
  const data = JSON.parse(payloadEl.textContent);
  const mountEl = container.querySelector(".codebox-editor");

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

  const displayLines = Math.min(
    Math.max(Number(data.maxLine) - Number(data.minLine) + 1, 1),
    20
  );

  requestAnimationFrame(() => {
    const lineHeight = view.defaultLineHeight;
    view.dom.style.height = Math.ceil(lineHeight * displayLines) + "px";
    view.requestMeasure();

    const midLine = Math.round((Number(data.minLine) + Number(data.maxLine)) / 2);
    const clampedMidLine = Math.min(Math.max(midLine, 1), doc.lines);
    const midPos = doc.line(clampedMidLine).from;
    view.dispatch({ effects: EditorView.scrollIntoView(midPos, { y: "center" }) });
  });

  return {
    boxId: data.boxId,
    resultIndex: data.resultIndex,
    view,
    containerEl: container,
    highlights: highlightRanges,
  };
}

const boxes = Array.from(document.querySelectorAll('[data-role="codebox"]')).map(mountBox);
initFlowArrows(boxes);
