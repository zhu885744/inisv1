import BaseAPI from './base'
import config from './config'

class API {
  constructor() {
    this.BaseAPI = BaseAPI
    this.config = config
  }
}

export default new API()