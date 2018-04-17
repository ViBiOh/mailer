import React, { Component } from 'react';
import { EditorState, ContentState } from 'draft-js';
import Prism from 'prismjs';
import Editor from 'draft-js-plugins-editor';
import createCodeEditorPlugin from 'draft-js-code-editor-plugin';
import createPrismPlugin from 'draft-js-prism-plugin';
import style from './index.css';

/**
 * Editor Component.
 */
export default class MailEditor extends Component {
  /**
   * Creates an instance of Editor.
   * @param {Object} props Component props
   */
  constructor(props) {
    super(props);

    this.state = {
      editorState: EditorState.createWithContent(
        ContentState.createFromText(`
<mjml>
  <mj-body>
    <mj-container>
      <mj-section>
        <mj-column>
          <mj-text>Hello {{ .Name }} !</mj-text>
        </mj-column>
      </mj-section>
    </mj-container>
  </mj-body>
</mjml>
`),
        this.decorator,
      ),
      plugins: [
        createCodeEditorPlugin(),
        createPrismPlugin({
          prism: Prism,
        }),
      ],
    };

    this.onChange = this.onChange.bind(this);
  }

  onChange(editorState) {
    this.setState({ editorState });
  }

  /**
   * React lifecycle.
   */
  render() {
    return (
      <div className={style.editor}>
        <Editor
          editorState={this.state.editorState}
          plugins={this.state.plugins}
          onChange={this.onChange}
        />
        <div>Output goes here</div>
      </div>
    );
  }
}
