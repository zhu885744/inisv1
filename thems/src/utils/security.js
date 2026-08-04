/**
 * XSS 防护工具
 * 提供 HTML 清理、字符串转义等安全功能
 */

/**
 * HTML 字符转义
 * @param {string} str - 要转义的字符串
 * @returns {string} 转义后的字符串
 */
export const escapeHtml = (str) => {
  if (!str || typeof str !== 'string') {
    return str
  }
  
  const escapeMap = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
    '/': '&#x2F;',
    '`': '&#x60;',
    '=': '&#x3D;'
  }
  
  return str.replace(/[&<>"'`=/]/g, (char) => escapeMap[char] || char)
}

/**
 * HTML 字符反转义
 * @param {string} str - 要反转义的字符串
 * @returns {string} 反转义后的字符串
 */
export const unescapeHtml = (str) => {
  if (!str || typeof str !== 'string') {
    return str
  }
  
  const unescapeMap = {
    '&amp;': '&',
    '&lt;': '<',
    '&gt;': '>',
    '&quot;': '"',
    '&#39;': "'",
    '&#x2F;': '/',
    '&#x60;': '`',
    '&#x3D;': '='
  }
  
  return str.replace(/&(amp|lt|gt|quot|#39|#x2F|#x60|#x3D);/g, (match, entity) => {
    return unescapeMap[match] || match
  })
}

/**
 * 清理 HTML 标签
 * @param {string} html - HTML 字符串
 * @param {object} options - 配置选项
 * @param {string[]} options.allowedTags - 允许的标签列表
 * @param {string[]} options.allowedAttrs - 允许的属性列表
 * @returns {string} 清理后的字符串
 */
export const sanitizeHtml = (html, options = {}) => {
  if (!html || typeof html !== 'string') {
    return html
  }
  
  const {
    allowedTags = ['b', 'i', 'u', 'em', 'strong', 'br', 'p', 'a', 'img'],
    allowedAttrs = ['href', 'src', 'alt', 'title']
  } = options
  
  let result = html
  
  result = result.replace(/<script[\s\S]*?<\/script>/gi, '')
  result = result.replace(/<style[\s\S]*?<\/style>/gi, '')
  result = result.replace(/<iframe[\s\S]*?<\/iframe>/gi, '')
  result = result.replace(/<object[\s\S]*?<\/object>/gi, '')
  result = result.replace(/<embed[\s\S]*?<\/embed>/gi, '')
  
  result = result.replace(/<(\/?)([a-z][a-z0-9]*)([^>]*)>/gi, (match, closing, tagName, attrs) => {
    const lowerTagName = tagName.toLowerCase()
    
    if (!allowedTags.includes(lowerTagName)) {
      return escapeHtml(match)
    }
    
    let cleanedAttrs = ''
    if (attrs) {
      attrs.replace(/([a-z][a-z0-9_]*)\s*=\s*["']([^"']*)["']/gi, (attrMatch, attrName, attrValue) => {
        const lowerAttrName = attrName.toLowerCase()
        
        if (allowedAttrs.includes(lowerAttrName)) {
          if (lowerAttrName === 'href' || lowerAttrName === 'src') {
            if (attrValue.startsWith('http://') || attrValue.startsWith('https://')) {
              cleanedAttrs += ` ${attrName}="${escapeHtml(attrValue)}"`
            }
          } else {
            cleanedAttrs += ` ${attrName}="${escapeHtml(attrValue)}"`
          }
        }
        
        return ''
      })
    }
    
    return `<${closing}${lowerTagName}${cleanedAttrs}>`
  })
  
  return result
}

/**
 * 检测 XSS 攻击
 * @param {string} str - 要检测的字符串
 * @returns {boolean} 是否包含 XSS 攻击
 */
export const detectXSS = (str) => {
  if (!str || typeof str !== 'string') {
    return false
  }
  
  const xssPatterns = [
    /<script[^>]*>[\s\S]*?<\/script>/i,
    /javascript:/i,
    /vbscript:/i,
    /data:/i,
    /onerror\s*=/i,
    /onload\s*=/i,
    /onclick\s*=/i,
    /onmouseover\s*=/i,
    /eval\s*\(/i,
    /document\.cookie/i,
    /document\.write/i,
    /window\.location/i,
    /<iframe/i,
    /<object/i,
    /<embed/i,
    /<svg/i,
    /<form/i,
    /<input/i,
    /<textarea/i,
    /<link/i,
    /<style/i,
    /alert\s*\(/i,
    /confirm\s*\(/i,
    /prompt\s*\(/i,
    /setTimeout\s*\(/i,
    /setInterval\s*\(/i
  ]
  
  return xssPatterns.some(pattern => pattern.test(str))
}

/**
 * 验证 URL 安全性
 * @param {string} url - 要验证的 URL
 * @returns {boolean} 是否为安全 URL
 */
export const isSafeUrl = (url) => {
  if (!url || typeof url !== 'string') {
    return false
  }
  
  try {
    const parsedUrl = new URL(url)
    
    if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
      return false
    }
    
    if (parsedUrl.hostname.includes('localhost')) {
      return true
    }
    
    const allowedDomains = ['zhuxu.asia', 'cs.zhuxu.asia']
    return allowedDomains.some(domain => 
      parsedUrl.hostname === domain || parsedUrl.hostname.endsWith(`.${domain}`)
    )
    
  } catch {
    return false
  }
}

/**
 * 验证邮箱格式
 * @param {string} email - 邮箱地址
 * @returns {boolean} 是否有效
 */
export const isValidEmail = (email) => {
  if (!email || typeof email !== 'string') {
    return false
  }
  
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}

/**
 * 验证手机号格式（中国）
 * @param {string} phone - 手机号
 * @returns {boolean} 是否有效
 */
export const isValidPhone = (phone) => {
  if (!phone || typeof phone !== 'string') {
    return false
  }
  
  return /^1[3-9]\d{9}$/.test(phone.replace(/\s/g, ''))
}

/**
 * 验证密码强度
 * @param {string} password - 密码
 * @returns {number} 强度等级 (0-4)
 */
export const getPasswordStrength = (password) => {
  if (!password || typeof password !== 'string') {
    return 0
  }
  
  let score = 0
  
  if (password.length >= 8) score++
  if (password.length >= 12) score++
  if (/[a-z]/.test(password)) score++
  if (/[A-Z]/.test(password)) score++
  if (/[0-9]/.test(password)) score++
  if (/[^a-zA-Z0-9]/.test(password)) score++
  
  return Math.min(4, Math.floor(score / 1.5))
}

/**
 * 生成安全随机字符串
 * @param {number} length - 长度
 * @returns {string} 随机字符串
 */
export const generateSecureRandom = (length = 32) => {
  const charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) {
    const array = new Uint32Array(length)
    crypto.getRandomValues(array)
    for (let i = 0; i < length; i++) {
      result += charset[array[i] % charset.length]
    }
  } else {
    for (let i = 0; i < length; i++) {
      result += charset[Math.floor(Math.random() * charset.length)]
    }
  }
  
  return result
}

/**
 * 安全存储数据到 localStorage
 * @param {string} key - 存储键
 * @param {any} value - 存储值
 * @param {number} ttl - 过期时间（分钟）
 */
export const safeLocalStorageSet = (key, value, ttl = 0) => {
  try {
    if (!key || typeof key !== 'string') {
      throw new Error('Invalid storage key')
    }
    
    if (detectXSS(JSON.stringify(value))) {
      throw new Error('Potential XSS detected in storage value')
    }
    
    const data = {
      value: value,
      timestamp: Date.now(),
      ttl: ttl * 60 * 1000
    }
    
    localStorage.setItem(key, JSON.stringify(data))
    
  } catch (error) {
    console.error('[Security] localStorage error:', error.message)
  }
}

/**
 * 安全从 localStorage 获取数据
 * @param {string} key - 存储键
 * @returns {any} 存储值
 */
export const safeLocalStorageGet = (key) => {
  try {
    if (!key || typeof key !== 'string') {
      return null
    }
    
    const rawData = localStorage.getItem(key)
    if (!rawData) {
      return null
    }
    
    const data = JSON.parse(rawData)
    
    if (data.ttl > 0 && Date.now() > data.timestamp + data.ttl) {
      localStorage.removeItem(key)
      return null
    }
    
    return data.value
    
  } catch (error) {
    console.error('[Security] localStorage error:', error.message)
    return null
  }
}

export default {
  escapeHtml,
  unescapeHtml,
  sanitizeHtml,
  detectXSS,
  isSafeUrl,
  isValidEmail,
  isValidPhone,
  getPasswordStrength,
  generateSecureRandom,
  safeLocalStorageSet,
  safeLocalStorageGet
}