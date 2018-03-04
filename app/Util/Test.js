/* eslint-disable import/no-extraneous-dependencies */
import { JSDOM } from 'jsdom';
import { configure } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';

const DEFAULT_HTML = '<html><body></body></html>';
global.document = new JSDOM(DEFAULT_HTML).window.document;
global.window = document.defaultView;
global.navigator = window.navigator;

configure({ adapter: new Adapter() });

global.then = (callback, timeout = 4) =>
  new Promise(resolve => setTimeout(resolve, timeout)).then(callback);
