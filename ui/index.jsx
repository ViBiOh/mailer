import React from 'react';
import ReactDOM from 'react-dom';
import { init } from './Constants';
import MailEditor from './MailEditor';

init();

ReactDOM.render(<MailEditor />, document.getElementById('root'));
