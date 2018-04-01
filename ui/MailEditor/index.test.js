import React from 'react';
import test from 'ava';
import { shallow } from 'enzyme';
import Editor from './';

function defaultProps() {
  return {};
}

test('should always render as a div', (t) => {
  const props = defaultProps();
  const wrapper = shallow(<Editor {...props} />);
  t.is(wrapper.type(), 'div');
});
