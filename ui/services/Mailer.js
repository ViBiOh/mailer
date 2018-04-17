import funtch from 'funtch';
import { getAPI } from '../Constants';

export default class Mailer {
  /**
   * List availables templates
   * @return {Promise} Result
   */
  static listTemplates() {
    return funtch.get(`${getAPI()}/render/`);
  }
}
