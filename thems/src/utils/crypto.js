import CryptoJS from 'crypto-js'

/**
 * 加密工具类
 * 包含：AES加密、MD5签名、Token生成、随机字符串等功能
 * 支持：环境变量配置、多种加密算法、错误处理
 */

// 默认配置（从环境变量读取，避免硬编码密钥）
const DEFAULT_CONFIG = {
  AES_KEY: import.meta.env.VITE_AES_KEY || '',
  AES_IV: import.meta.env.VITE_AES_IV || '',
  
  TOKEN_MIN_LENGTH: 8,
  TOKEN_MAX_LENGTH: 64,
  DEFAULT_TOKEN_LENGTH: 32,
  
  SALT: import.meta.env.VITE_CRYPTO_SALT || ''
}

/**
 * AES-CBC对称加密类
 */
class AESCrypto {
  /**
   * 构造函数
   * @param {string} key - 加密密钥（16/24/32位）
   * @param {string} iv - 加密向量（16位）
   */
  constructor(key = DEFAULT_CONFIG.AES_KEY, iv = DEFAULT_CONFIG.AES_IV) {
    this.validateCryptoJS()
    
    if (!key || !iv) {
      console.warn('AES密钥或向量未配置，加密功能不可用')
    }
    
    this.originalKey = this.normalizeKey(key, DEFAULT_CONFIG.AES_KEY);
    this.originalIv = this.normalizeIv(iv, DEFAULT_CONFIG.AES_IV);
    
    // 解析为CryptoJS格式
    this.key = CryptoJS.enc.Utf8.parse(this.originalKey)
    this.iv = CryptoJS.enc.Utf8.parse(this.originalIv)
    
    // 验证向量长度（已强制处理，仅调试提示）
    this.validateIvLength()
  }

  // 规范化IV为16位
  normalizeIv(iv, defaultValue) {
    let normalized = this.normalizeKey(iv, defaultValue);
    // 截断到16位或补全到16位
    if (normalized.length > 16) {
      normalized = normalized.substring(0, 16);
    } else if (normalized.length < 16) {
      normalized = normalized.padEnd(16, '0'); // 不足补0
    }
    return normalized;
  }

  /**
   * 验证CryptoJS是否可用
   */
  validateCryptoJS() {
    if (!CryptoJS || !CryptoJS.AES || !CryptoJS.enc) {
      throw new Error('CryptoJS加载失败，请检查依赖')
    }
  }

  /**
   * 规范化密钥
   */
  normalizeKey(key, defaultValue) {
    if (typeof key !== 'string' || key.trim() === '') {
      return defaultValue
    }
    return key.trim()
  }

  /**
   * 验证向量长度（调试提示）
   */
  validateIvLength() {
    if (this.originalIv.length !== 16) {
      console.debug(`AES IV已自动调整为16位，原始长度：${this.originalIv.length}`);
    }
  }

  /**
   * 加密
   * @param {any} data - 要加密的数据
   * @returns {string} Base64加密字符串
   */
  encrypt(data) {
    try {
      // 转换为字符串
      const text = this.dataToString(data)
      if (!text) return ''
      
      // 执行加密
      const encrypted = CryptoJS.AES.encrypt(text, this.key, {
        iv: this.iv,
        mode: CryptoJS.mode.CBC,
        padding: CryptoJS.pad.Pkcs7
      })
      
      return encrypted.toString()
    } catch (error) {
      console.error('AES加密失败:', error.message)
      return ''
    }
  }

  /**
   * 解密
   * @param {string} encryptedText - 加密的Base64字符串
   * @returns {any} 解密后的数据
   */
  decrypt(encryptedText) {
    try {
      if (!encryptedText || typeof encryptedText !== 'string') {
        return ''
      }
      
      // 执行解密
      const decrypted = CryptoJS.AES.decrypt(encryptedText, this.key, {
        iv: this.iv,
        mode: CryptoJS.mode.CBC,
        padding: CryptoJS.pad.Pkcs7
      })
      
      // 转换为字符串
      const text = decrypted.toString(CryptoJS.enc.Utf8)
      
      // 尝试解析为JSON
      return this.parseDecryptedText(text)
    } catch (error) {
      console.error('AES解密失败:', error.message)
      return ''
    }
  }

  /**
   * 数据转字符串
   */
  dataToString(data) {
    if (data === null || data === undefined) return ''
    if (typeof data === 'string') return data
    if (typeof data === 'number' || typeof data === 'boolean') return String(data)
    
    try {
      return JSON.stringify(data)
    } catch {
      return String(data)
    }
  }

  /**
   * 解析解密后的文本
   */
  parseDecryptedText(text) {
    if (!text) return ''
    
    try {
      // 尝试解析JSON
      return JSON.parse(text)
    } catch {
      // 返回原始字符串
      return text
    }
  }

  /**
   * 获取配置信息
   */
  getConfig() {
    return {
      keyLength: this.originalKey.length,
      ivLength: this.originalIv.length,
      algorithm: 'AES-CBC-PKCS7'
    }
  }
}

/**
 * 哈希工具类
 */
class Hash {
  /**
   * 生成MD5哈希
   */
  static md5(data, salt = '') {
    try {
      const text = Hash.dataToString(data) + salt
      return CryptoJS.MD5(text).toString()
    } catch (error) {
      console.error('MD5生成失败:', error.message)
      return ''
    }
  }

  /**
   * 生成SHA256哈希（修复：方法名小写，符合JS规范）
   */
  static sha256(data, salt = '') {
    try {
      const text = Hash.dataToString(data) + salt
      return CryptoJS.SHA256(text).toString()
    } catch (error) {
      console.error('SHA256生成失败:', error.message)
      return ''
    }
  }

  /**
   * 生成SHA1哈希
   */
  static sha1(data, salt = '') {
    try {
      const text = Hash.dataToString(data) + salt
      return CryptoJS.SHA1(text).toString()
    } catch (error) {
      console.error('SHA1生成失败:', error.message)
      return ''
    }
  }

  /**
   * 生成HMAC-SHA256签名
   */
  static hmac(data, secret) {
    try {
      const text = Hash.dataToString(data)
      return CryptoJS.HmacSHA256(text, secret).toString()
    } catch (error) {
      console.error('HMAC生成失败:', error.message)
      return ''
    }
  }

  /**
   * 数据转字符串
   */
  static dataToString(data) {
    if (data === null || data === undefined) return ''
    if (typeof data === 'string') return data
    if (typeof data === 'number' || typeof data === 'boolean') return String(data)
    
    try {
      return JSON.stringify(data)
    } catch {
      return String(data)
    }
  }
}

/**
 * Token生成器
 */
class TokenGenerator {
  /**
   * 生成指定长度的Token
   */
  static generate(value = '', length = DEFAULT_CONFIG.DEFAULT_TOKEN_LENGTH, options = {}) {
    try {
      // 验证参数
      const config = this.validateOptions(length, options)
      
      // 生成基础哈希
      let token = this.generateBaseToken(value, config)
      
      // 处理长度
      token = this.adjustTokenLength(token, config.length)
      
      // 格式化输出
      return this.formatToken(token, config)
    } catch (error) {
      console.error('Token生成失败:', error.message)
      return ''
    }
  }

  /**
   * 验证和规范化选项
   */
  static validateOptions(length, options) {
    const {
      prefix = '',
      salt = DEFAULT_CONFIG.SALT,
      algorithm = 'md5',
      includeTimestamp = false,
      includeRandom = false,
      uppercase = false,
      specialChars = false
    } = options

    // 验证长度
    const validLength = Math.max(
      DEFAULT_CONFIG.TOKEN_MIN_LENGTH,
      Math.min(DEFAULT_CONFIG.TOKEN_MAX_LENGTH, Number(length) || DEFAULT_CONFIG.DEFAULT_TOKEN_LENGTH)
    )

    return {
      length: validLength,
      prefix: String(prefix),
      salt: String(salt),
      algorithm,
      includeTimestamp,
      includeRandom,
      uppercase,
      specialChars
    }
  }

  /**
   * 生成基础Token
   */
  static generateBaseToken(value, config) {
    // 准备数据
    let data = this.prepareData(value, config)
    
    // 根据算法生成哈希
    switch (config.algorithm.toLowerCase()) {
      case 'sha256':
        return Hash.sha256(data, config.salt)
      case 'sha1':
        return Hash.sha1(data, config.salt)
      case 'md5':
      default:
        return Hash.md5(data, config.salt)
    }
  }

  /**
   * 准备数据
   */
  static prepareData(value, config) {
    let data = ''
    
    // 添加前缀
    if (config.prefix) {
      data += config.prefix
    }
    
    // 添加原始值
    data += Hash.dataToString(value)
    
    // 添加时间戳
    if (config.includeTimestamp) {
      data += Date.now().toString()
    }
    
    // 添加随机数
    if (config.includeRandom) {
      data += Math.random().toString(36).substr(2, 9)
    }
    
    return data
  }

  /**
   * 调整Token长度
   */
  static adjustTokenLength(token, targetLength) {
    if (token.length >= targetLength) {
      return token.substring(0, targetLength)
    }
    
    // 长度不足，进行扩展
    let result = token
    while (result.length < targetLength) {
      const remaining = targetLength - result.length
      const extension = Hash.md5(result + Date.now()).substring(0, remaining)
      result += extension
    }
    
    return result.substring(0, targetLength)
  }

  /**
   * 格式化Token
   */
  static formatToken(token, config) {
    let formatted = token
    
    // 大写转换
    if (config.uppercase) {
      formatted = formatted.toUpperCase()
    }
    
    // 特殊字符（如果需要）
    if (config.specialChars) {
      formatted = this.addSpecialChars(formatted)
    }
    
    return formatted
  }

  /**
   * 添加特殊字符
   */
  static addSpecialChars(token) {
    const specialChars = '!@#$%^&*'
    let result = ''
    let charIndex = 0
    
    for (let i = 0; i < token.length; i++) {
      result += token[i]
      if (i > 0 && i % 4 === 0 && charIndex < specialChars.length) {
        result += specialChars[charIndex++]
      }
    }
    
    return result
  }

  /**
   * 生成随机Token
   */
  static random(length = 32) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
    let result = ''
    
    const validLength = Math.max(8, Math.min(64, Number(length) || 32))
    
    for (let i = 0; i < validLength; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length))
    }
    
    return result
  }

  /**
   * 生成UUID
   */
  static uuid() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
      const r = Math.random() * 16 | 0
      const v = c === 'x' ? r : (r & 0x3 | 0x8)
      return v.toString(16)
    })
  }

  /**
   * 验证Token格式
   */
  static validate(token, minLength = 8, maxLength = 64) {
    if (!token || typeof token !== 'string') return false
    
    const length = token.length
    if (length < minLength || length > maxLength) return false
    
    // 可以添加更多验证规则
    return /^[a-zA-Z0-9!@#$%^&*\-_+=]*$/.test(token)
  }
}

/**
 * 随机字符串生成器
 */
class Random {
  /**
   * 生成随机字符串
   */
  static string(length = 16, options = {}) {
    const {
      uppercase = true,
      lowercase = true,
      numbers = true,
      special = false
    } = options

    let chars = ''
    if (uppercase) chars += 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
    if (lowercase) chars += 'abcdefghijklmnopqrstuvwxyz'
    if (numbers) chars += '0123456789'
    if (special) chars += '!@#$%^&*'

    if (!chars) chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'

    let result = ''
    for (let i = 0; i < length; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length))
    }

    return result
  }

  /**
   * 生成随机数字
   */
  static number(min = 0, max = 100) {
    return Math.floor(Math.random() * (max - min + 1)) + min
  }

  /**
   * 生成随机字节（浏览器环境）
   */
  static bytes(length = 16) {
    if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
      const array = new Uint8Array(length)
      crypto.getRandomValues(array)
      return Array.from(array).map(b => b.toString(16).padStart(2, '0')).join('')
    } else {
      // 降级方案
      let result = ''
      for (let i = 0; i < length * 2; i++) {
        result += Math.floor(Math.random() * 16).toString(16)
      }
      return result
    }
  }
}

/**
 * 主加密工具类（重命名避免冲突）
 */
class CryptoUtil {
  constructor() {
    this.defaultAES = new AESCrypto()
  }

  /**
   * 创建AES实例
   */
  createAES(key, iv) {
    return new AESCrypto(key, iv)
  }

  /**
   * Token生成（兼容旧API）
   */
  token(value = '', length = 32, prefix = '') {
    return TokenGenerator.generate(value, length, { prefix })
  }

  /**
   * 快捷方法
   */
  get aes() {
    return this.defaultAES
  }

  get hash() {
    return Hash
  }

  get tokenGen() {
    return TokenGenerator
  }

  get random() {
    return Random
  }

  /**
   * 静态方法
   */
  static get AES() {
    return AESCrypto
  }

  static get Hash() {
    return Hash
  }

  static get Token() {
    return TokenGenerator
  }

  static get Random() {
    return Random
  }

  /**
   * 快速生成随机Token
   */
  static randomToken(length = 32) {
    return TokenGenerator.random(length)
  }

  /**
   * 快速生成UUID
   */
  static uuid() {
    return TokenGenerator.uuid()
  }

  /**
   * 快速加密
   */
  static encrypt(data, key, iv) {
    const aes = new AESCrypto(key, iv)
    return aes.encrypt(data)
  }

  /**
   * 快速解密
   */
  static decrypt(encrypted, key, iv) {
    const aes = new AESCrypto(key, iv)
    return aes.decrypt(encrypted)
  }
}

// 创建默认实例（重命名为 cryptoUtil，避免与原生 crypto 冲突）
const cryptoUtil = new CryptoUtil()

export default cryptoUtil

// 导出所有工具类
export {
  AESCrypto,
  Hash,
  TokenGenerator,
  Random,
  CryptoUtil
}