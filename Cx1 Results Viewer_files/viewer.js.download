import { EditorView, basicSetup } from "codemirror";
import { EditorState, Text } from "@codemirror/state";
import { Decoration } from "@codemirror/view";

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
  const { from, to } = computeRange(doc, data.line, data.column, data.length);

  const highlightExtension = to > from
    ? EditorView.decorations.of(
        Decoration.set([Decoration.mark({ class: "cm-node-highlight" }).range(from, to)])
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
    const lineHeight = view.defaultLineHeight;
    view.dom.style.height = Math.ceil(lineHeight * 5) + "px";
    view.requestMeasure();
    view.dispatch({ effects: EditorView.scrollIntoView(from, { y: "center" }) });
  });
}

document.querySelectorAll('[data-role="codebox"]').forEach(mountBox);
