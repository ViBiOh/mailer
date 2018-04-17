import funtch from 'funtch';

let context = {};

/**
 * Initialize config from /env endpoint
 * @return {Promise} Config object
 */
export function init() {
  return new Promise((resolve) => {
    funtch.get('/env').then((env) => {
      context = env;
      resolve(context);
    });
  });
}

/**
 * Return API URL.
 * @return {String} API Base URL
 */
export function getAPI() {
  return context.API_URL || 'https://mailer-api.vibioh.fr';
}
