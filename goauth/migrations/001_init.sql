-- Goauth 数据库初始化
-- SQLite3

-- 用户表
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT UNIQUE,
  username TEXT UNIQUE NOT NULL,
  name TEXT,
  passwordHash TEXT,
  isAdmin INTEGER DEFAULT 0,
  emailVerified INTEGER DEFAULT 0,
  approved INTEGER DEFAULT 0,
  mfaRequired INTEGER DEFAULT 0,
  disabled INTEGER DEFAULT 0,
  createdAt TEXT NOT NULL,
  updatedAt TEXT NOT NULL
);

-- Session表
CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  userId TEXT NOT NULL,
  token TEXT NOT NULL UNIQUE,
  amr TEXT,
  rememberMe INTEGER DEFAULT 0,
  expiresAt TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  FOREIGN KEY (userId) REFERENCES users(id) ON DELETE CASCADE
);

-- 分组表
CREATE TABLE IF NOT EXISTS groups (
  id TEXT PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  mfaRequired INTEGER DEFAULT 0,
  createdBy TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  updatedAt TEXT NOT NULL,
  FOREIGN KEY (createdBy) REFERENCES users(id)
);

-- 用户-分组关联
CREATE TABLE IF NOT EXISTS user_groups (
  userId TEXT NOT NULL,
  groupId TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  PRIMARY KEY (userId, groupId),
  FOREIGN KEY (userId) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (groupId) REFERENCES groups(id) ON DELETE CASCADE
);

-- TOTP配置
CREATE TABLE IF NOT EXISTS totp (
  id TEXT PRIMARY KEY,
  userId TEXT NOT NULL UNIQUE,
  secret TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  updatedAt TEXT NOT NULL,
  FOREIGN KEY (userId) REFERENCES users(id) ON DELETE CASCADE
);

-- 密钥存储
CREATE TABLE IF NOT EXISTS keys (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  value TEXT NOT NULL,
  expiresAt TEXT NOT NULL,
  createdAt TEXT NOT NULL
);

-- OIDC Payloads
CREATE TABLE IF NOT EXISTS oidc_payloads (
  id TEXT NOT NULL,
  type TEXT NOT NULL,
  payload TEXT NOT NULL,
  grantId TEXT,
  userCode TEXT,
  uid TEXT,
  expiresAt TEXT,
  consumedAt TEXT,
  accountId TEXT,
  PRIMARY KEY (id, type)
);

-- OIDC 客户端
CREATE TABLE IF NOT EXISTS clients (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  secret TEXT,
  redirectUris TEXT NOT NULL,
  postLogoutUris TEXT,
  scopes TEXT NOT NULL,
  grantTypes TEXT NOT NULL,
  responseTypes TEXT NOT NULL,
  tokenEndpointAuth TEXT DEFAULT 'client_secret_basic',
  trusted INTEGER DEFAULT 0,
  createdBy TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  updatedAt TEXT NOT NULL,
  FOREIGN KEY (createdBy) REFERENCES users(id)
);

-- Consent
CREATE TABLE IF NOT EXISTS consent (
  userId TEXT NOT NULL,
  clientId TEXT NOT NULL,
  scope TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  expiresAt TEXT NOT NULL,
  PRIMARY KEY (userId, clientId)
);

-- 邀请
CREATE TABLE IF NOT EXISTS invitations (
  id TEXT PRIMARY KEY,
  email TEXT,
  username TEXT,
  name TEXT,
  challenge TEXT NOT NULL,
  emailVerified INTEGER DEFAULT 0,
  createdBy TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  expiresAt TEXT NOT NULL,
  FOREIGN KEY (createdBy) REFERENCES users(id)
);

-- 邀请-分组关联
CREATE TABLE IF NOT EXISTS invitation_groups (
  invitationId TEXT NOT NULL,
  groupId TEXT NOT NULL,
  PRIMARY KEY (invitationId, groupId),
  FOREIGN KEY (invitationId) REFERENCES invitations(id) ON DELETE CASCADE,
  FOREIGN KEY (groupId) REFERENCES groups(id) ON DELETE CASCADE
);

-- ProxyAuth
CREATE TABLE IF NOT EXISTS proxy_auth (
  id TEXT PRIMARY KEY,
  domain TEXT NOT NULL UNIQUE,
  mfaRequired INTEGER DEFAULT 0,
  maxSessionLength INTEGER,
  createdBy TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  updatedAt TEXT NOT NULL,
  FOREIGN KEY (createdBy) REFERENCES users(id)
);

-- ProxyAuth-分组关联
CREATE TABLE IF NOT EXISTS proxy_auth_groups (
  proxyAuthId TEXT NOT NULL,
  groupId TEXT NOT NULL,
  PRIMARY KEY (proxyAuthId, groupId),
  FOREIGN KEY (proxyAuthId) REFERENCES proxy_auth(id) ON DELETE CASCADE,
  FOREIGN KEY (groupId) REFERENCES groups(id) ON DELETE CASCADE
);

-- 系统标志
CREATE TABLE IF NOT EXISTS flags (
  name TEXT PRIMARY KEY,
  value TEXT,
  createdAt TEXT NOT NULL
);

-- 登录尝试记录（暴力破解防护）
CREATE TABLE IF NOT EXISTS login_attempts (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL,
  ip TEXT NOT NULL,
  success INTEGER DEFAULT 0,
  createdAt TEXT NOT NULL
);

-- 审计日志
CREATE TABLE IF NOT EXISTS audit_log (
  id TEXT PRIMARY KEY,
  action TEXT NOT NULL,
  actorId TEXT,
  targetId TEXT,
  details TEXT,
  ip TEXT NOT NULL,
  createdAt TEXT NOT NULL,
  FOREIGN KEY (actorId) REFERENCES users(id) ON DELETE SET NULL
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_userId ON sessions(userId);
CREATE INDEX IF NOT EXISTS idx_sessions_expiresAt ON sessions(expiresAt);
CREATE INDEX IF NOT EXISTS idx_oidc_payloads_expiresAt ON oidc_payloads(expiresAt);
CREATE INDEX IF NOT EXISTS idx_oidc_payloads_accountId ON oidc_payloads(accountId);
CREATE INDEX IF NOT EXISTS idx_oidc_payloads_grantId ON oidc_payloads(grantId);
CREATE INDEX IF NOT EXISTS idx_oidc_payloads_uid ON oidc_payloads(uid);
CREATE INDEX IF NOT EXISTS idx_users_approved ON users(approved);
CREATE INDEX IF NOT EXISTS idx_users_disabled ON users(disabled);
CREATE INDEX IF NOT EXISTS idx_login_attempts_username ON login_attempts(username);
CREATE INDEX IF NOT EXISTS idx_login_attempts_ip ON login_attempts(ip);
CREATE INDEX IF NOT EXISTS idx_login_attempts_createdAt ON login_attempts(createdAt);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_log_createdAt ON audit_log(createdAt);
CREATE INDEX IF NOT EXISTS idx_keys_expiresAt ON keys(expiresAt);
