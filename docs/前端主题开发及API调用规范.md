# 前端主题开发及 API 调用规范

## 一、前端主题开发规范

### 1.1 主题目录结构

主题必须遵循以下目录结构：

```
themes/
├── [theme-name]/                    # 主题名称（小写英文，连字符分隔）
│   ├── src/                         # 源码目录
│   │   ├── components/              # 公共组件
│   │   │   ├── Header.vue           # 头部组件
│   │   │   ├── Footer.vue           # 底部组件
│   │   │   ├── Sidebar.vue          # 侧边栏组件
│   │   │   └── ...
│   │   ├── views/                   # 页面视图
│   │   │   ├── index.vue            # 首页
│   │   │   ├── article.vue          # 文章详情页
│   │   │   ├── category.vue         # 分类页
│   │   │   └── ...
│   │   ├── layouts/                 # 布局组件
│   │   │   ├── default.vue          # 默认布局
│   │   │   └── fullscreen.vue       # 全屏布局
│   │   ├── assets/                  # 静态资源
│   │   │   ├── css/                 # 样式文件
│   │   │   ├── js/                  # 自定义脚本
│   │   │   └── images/              # 图片资源
│   │   ├── utils/                   # 工具函数
│   │   │   └── api.js               # API 封装
│   │   └── config.js                # 主题配置
│   ├── package.json                 # 依赖配置
│   └── theme.config.js              # 主题元配置
```

### 1.2 组件命名规范

| 类型 | 命名规则 | 示例 |
|------|----------|------|
| 组件文件 | PascalCase | `HeaderNav.vue` |
| 组件名称 | PascalCase | `HeaderNav` |
| 组件内部变量 | camelCase | `navItems` |
| 组件props | camelCase | `articleId` |

### 1.3 样式规范

- **CSS 类命名**：采用 BEM 命名法
- **颜色变量**：统一在 `variables.scss` 中定义
- **响应式断点**：使用 Tailwind CSS 预设断点

```scss
// 示例：BEM 命名
.header {
  &__logo {}
  &__nav {}
  &__nav-item {
    &--active {}
  }
}
```

### 1.4 主题配置规范

主题配置文件 `config.js` 必须包含以下字段：

```javascript
export default {
  name: 'theme-name',
  version: '1.0.0',
  author: 'Author Name',
  description: '主题描述',
  thumbnail: '/path/to/thumbnail.png',
  settings: {
    primaryColor: '#3B82F6',
    accentColor: '#8B5CF6',
    darkMode: false,
    showSidebar: true,
    // ... 其他可配置项
  }
}
```

---

## 二、API 调用统一规范

### 2.1 基础配置

#### 2.1.1 域名配置

```javascript
// 开发环境
const baseURL = process.env.NODE_ENV === 'development' 
  ? 'http://localhost:8080/api' 
  : '/api';
```

#### 2.1.2 请求头配置

```javascript
const headers = {
  'Content-Type': 'application/json',
  'X-Requested-With': 'XMLHttpRequest'
};

// 携带 Token
if (localStorage.getItem('token')) {
  headers['Authorization'] = `Bearer ${localStorage.getItem('token')}`;
}
```

### 2.2 请求方法封装

```javascript
// api.js
import axios from 'axios';

const instance = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
});

// 请求拦截器
instance.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  error => Promise.reject(error)
);

// 响应拦截器
instance.interceptors.response.use(
  response => {
    const { code, msg, data } = response.data;
    if (code === 200) {
      return { success: true, data, msg };
    } else if (code === 401) {
      // 未登录处理
      localStorage.removeItem('token');
      window.location.href = '/login';
      return { success: false, data: null, msg };
    }
    return { success: false, data, msg };
  },
  error => {
    return { 
      success: false, 
      data: null, 
      msg: error.response?.data?.msg || '网络请求失败' 
    };
  }
);

export default instance;
```

### 2.3 接口调用规范

#### 2.3.1 用户相关接口

```javascript
// users.js
import api from './api';

export const userApi = {
  // 获取当前用户信息
  getCurrentUser() {
    return api.get('/users/one', {
      params: { id: localStorage.getItem('userId') }
    });
  },

  // 获取用户列表（分页）
  getUserList(params = {}) {
    return api.get('/users/all', {
      params: {
        page: 1,
        limit: 10,
        ...params
      }
    });
  },

  // 更新用户信息
  updateUser(data) {
    return api.put('/users/update', data);
  },

  // 修改密码
  changePassword(oldPwd, newPwd) {
    return api.put('/users/password', { oldPwd, newPwd });
  }
};
```

#### 2.3.2 文章相关接口

```javascript
// articles.js
import api from './api';

export const articleApi = {
  // 获取文章列表
  getArticles(params = {}) {
    return api.get('/articles/all', {
      params: {
        page: 1,
        limit: 10,
        order: 'create_time desc',
        ...params
      }
    });
  },

  // 获取单篇文章
  getArticle(id) {
    return api.get('/articles/one', {
      params: { id }
    });
  },

  // 创建文章
  createArticle(data) {
    return api.post('/articles/create', data);
  },

  // 更新文章
  updateArticle(id, data) {
    return api.put('/articles/update', { id, ...data });
  },

  // 删除文章
  deleteArticle(ids) {
    return api.delete('/articles/remove', {
      params: { ids: Array.isArray(ids) ? ids.join(',') : ids }
    });
  }
};
```

### 2.4 响应处理规范

#### 2.4.1 统一响应格式

所有接口响应遵循以下格式：

```json
{
  "code": 200,
  "msg": "操作成功",
  "data": {}
}
```

#### 2.4.2 状态码处理

| 状态码 | 处理方式 |
|--------|----------|
| 200 | 正常处理数据 |
| 201 | 操作已受理（如验证码发送） |
| 204 | 无数据，显示空状态 |
| 400 | 提示用户参数错误 |
| 401 | 跳转到登录页 |
| 403 | 提示无权限 |
| 404 | 提示资源不存在 |
| 500 | 提示服务器错误 |

#### 2.4.3 响应处理示例

```javascript
async function fetchUserList() {
  try {
    const response = await userApi.getUserList({ page: 1 });
    if (response.success) {
      // 成功处理
      renderUserList(response.data);
    } else {
      // 失败处理
      showToast(response.msg);
    }
  } catch (error) {
    showToast('请求失败，请稍后重试');
  }
}
```

---

## 三、数据脱敏处理规范

### 3.1 敏感信息识别

前端需要处理的敏感信息包括：

| 字段类型 | 示例 | 脱敏规则 |
|----------|------|----------|
| 邮箱 | user@example.com | us***@example.com |
| 手机号 | 13812345678 | 138****5678 |
| 账号 | username123 | us********** |

### 3.2 脱敏工具函数

```javascript
// utils/mask.js
export const maskUtils = {
  // 邮箱脱敏
  maskEmail(email) {
    if (!email || !email.includes('@')) return email;
    const [prefix, domain] = email.split('@');
    if (prefix.length <= 2) return email;
    return `${prefix.slice(0, 2)}${'*'.repeat(prefix.length - 2)}@${domain}`;
  },

  // 手机号脱敏
  maskPhone(phone) {
    if (!phone) return phone;
    const cleaned = phone.replace(/[\s-]/g, '');
    if (cleaned.length !== 11) return phone;
    return `${cleaned.slice(0, 3)}****${cleaned.slice(7)}`;
  },

  // 账号脱敏
  maskAccount(account) {
    if (!account || account.length <= 2) return account;
    return `${account.slice(0, 2)}${'*'.repeat(account.length - 2)}`;
  }
};
```

### 3.3 脱敏使用场景

```javascript
// 使用示例
import { maskUtils } from '@/utils/mask';

// 显示用户信息时脱敏
const user = await userApi.getUserInfo(userId);
if (user.success) {
  const userData = user.data;
  userData.email = maskUtils.maskEmail(userData.email);
  userData.phone = maskUtils.maskPhone(userData.phone);
  renderUserInfo(userData);
}
```

---

## 四、错误处理规范

### 4.1 全局错误处理

```javascript
// 在响应拦截器中统一处理
instance.interceptors.response.use(
  response => response,
  error => {
    const status = error.response?.status;
    const msg = error.response?.data?.msg || '网络请求失败';

    switch (status) {
      case 400:
        showToast(`请求错误：${msg}`);
        break;
      case 401:
        showToast('请先登录');
        setTimeout(() => {
          window.location.href = '/login';
        }, 1500);
        break;
      case 403:
        showToast('暂无权限');
        break;
      case 404:
        showToast('资源不存在');
        break;
      case 500:
        showToast('服务器繁忙，请稍后重试');
        break;
      default:
        showToast(msg);
    }

    return Promise.reject(error);
  }
);
```

### 4.2 请求重试机制

```javascript
// 带重试的请求封装
async function requestWithRetry(fn, maxRetries = 3, delay = 1000) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const result = await fn();
      if (result.success) return result;
      throw new Error(result.msg);
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      await new Promise(resolve => setTimeout(resolve, delay * Math.pow(2, i)));
    }
  }
}
```

---

## 五、性能优化规范

### 5.1 请求缓存

```javascript
// 简单的请求缓存
const cache = new Map();

async function fetchWithCache(key, fetcher) {
  if (cache.has(key)) {
    return cache.get(key);
  }
  
  const result = await fetcher();
  cache.set(key, result);
  
  // 5分钟后过期
  setTimeout(() => {
    cache.delete(key);
  }, 5 * 60 * 1000);
  
  return result;
}
```

### 5.2 请求防抖/节流

```javascript
// 搜索防抖
function debounce(fn, delay = 300) {
  let timer = null;
  return function(...args) {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => fn.apply(this, args), delay);
  };
}

// 使用示例
const searchInput = document.querySelector('#search');
searchInput.addEventListener('input', debounce(async (e) => {
  const keyword = e.target.value;
  if (keyword.length >= 2) {
    await searchArticles(keyword);
  }
}, 300));
```

---

## 六、安全规范

### 6.1 XSS 防护

- 使用框架内置的 HTML 转义（如 Vue 的 `v-text`）
- 对用户输入进行过滤
- 使用 `DOMPurify` 清理富文本内容

### 6.2 CSRF 防护

- 使用 Token 验证
- 设置 `SameSite` Cookie 属性

### 6.3 请求安全

```javascript
// 防止请求篡改
const instance = axios.create({
  // ...
});

// 请求签名（可选）
instance.interceptors.request.use(config => {
  const timestamp = Date.now().toString();
  const signature = generateSignature(config, timestamp);
  config.headers['X-Signature'] = signature;
  config.headers['X-Timestamp'] = timestamp;
  return config;
});
```

---

## 七、开发工具链

### 7.1 推荐依赖

| 依赖 | 用途 | 版本 |
|------|------|------|
| axios | HTTP 请求 | ^1.0.0 |
| pinia | 状态管理 | ^2.0.0 |
| vue-router | 路由管理 | ^4.0.0 |
| lucide-vue-next | 图标库 | ^0.20.0 |
| dayjs | 日期处理 | ^1.0.0 |

### 7.2 代码格式化

- 使用 Prettier 统一代码风格
- 配置 `.prettierrc`：

```json
{
  "printWidth": 120,
  "tabWidth": 2,
  "useTabs": false,
  "semi": true,
  "singleQuote": true,
  "trailingComma": "es5"
}
```

---

## 八、版本控制规范

### 8.1 Git 分支管理

| 分支类型 | 命名规则 | 用途 |
|----------|----------|------|
| 主分支 | main | 生产环境 |
| 开发分支 | develop | 开发整合 |
| 功能分支 | feature/xxx | 新功能开发 |
| 修复分支 | bugfix/xxx | Bug 修复 |

### 8.2 Commit 信息规范

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Type 类型**：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式
- `refactor`: 重构
- `test`: 测试
- `chore`: 构建/工具

---

## 九、总结

1. **主题开发**：遵循统一目录结构和命名规范
2. **API 调用**：使用封装的 axios 实例，统一处理响应
3. **数据脱敏**：对敏感信息进行脱敏处理后展示
4. **错误处理**：统一拦截和处理各类错误状态码
5. **性能优化**：实现请求缓存和防抖节流
6. **安全防护**：注意 XSS、CSRF 防护

以上规范旨在提高代码质量、降低维护成本、保障系统安全。