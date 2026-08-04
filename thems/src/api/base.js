import { request } from '@/utils/network'

class BaseAPI {
  constructor(controller) {
    this.controller = controller
  }

  async all(params = {}) {
    return await request.get(`/api/${this.controller}/all`, { params })
  }

  async one(params = {}) {
    return await request.get(`/api/${this.controller}/one`, { params })
  }

  async count(params = {}) {
    return await request.get(`/api/${this.controller}/count`, { params })
  }

  async column(params = {}) {
    return await request.get(`/api/${this.controller}/column`, { params })
  }

  async create(data = {}) {
    return await request.post(`/api/${this.controller}/create`, data)
  }

  async update(data = {}) {
    return await request.post(`/api/${this.controller}/update`, data)
  }

  async save(data = {}) {
    return await request.post(`/api/${this.controller}/save`, data)
  }

  async remove(id) {
    return await request.post(`/api/${this.controller}/remove`, { id })
  }

  async delete(id) {
    return await request.post(`/api/${this.controller}/delete`, { id })
  }

  async clear(params = {}) {
    return await request.post(`/api/${this.controller}/clear`, params)
  }

  async restore(id) {
    return await request.post(`/api/${this.controller}/restore`, { id })
  }
}

export default BaseAPI