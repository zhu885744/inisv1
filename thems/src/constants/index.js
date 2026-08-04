export const STORAGE_KEYS = {
  DARK_MODE: 'dark-mode',
  USER_INFO: 'user-info',
  UID: 'uid',
  SEARCH_HISTORY: 'search-history',
  ROUTER_HISTORY: 'router_history',
  LAST_COMMENT_TIME: 'lastCommentTime',
  LAST_USER_ACTIVITY: 'lastUserActivity'
}

export const MESSAGE_TYPES = {
  SUCCESS: 'success',
  ERROR: 'error',
  WARNING: 'warning',
  INFO: 'info'
}

export const VALIDATION_RULES = {
  COMMENT: {
    MIN_LENGTH: 1,
    MAX_LENGTH: 1000,
    RATE_LIMIT: 30
  },
  USERNAME: {
    MIN_LENGTH: 2,
    MAX_LENGTH: 20
  },
  PASSWORD: {
    MIN_LENGTH: 6,
    MAX_LENGTH: 32
  }
}
