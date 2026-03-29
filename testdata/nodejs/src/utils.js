import _ from 'lodash';

export function mergeConfigs(...configs) {
  return _.merge({}, ...configs);
}

export function capitalize(str) {
  return _.capitalize(str);
}
