import React, { Component } from 'react';
import { Editor, EditorState, ContentState } from 'draft-js';

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
        ContentState.createFromText('Hello World !'),
        this.decorator,
      ),
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
      <div id="live-editor">
        <Editor editorState={this.state.editorState} onChange={this.onChange} />
        <div>
          Output goes here
        </div>
      </div>
    );
  }
}
