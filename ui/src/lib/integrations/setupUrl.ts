export interface SetupQueryParams {
  tool: string
  key: string
  serverUrl: string
  provider?: string
  model?: string
  models?: string[]
  subagentModel?: string
  haiku?: string
  sonnet?: string
  opus?: string
  telegramBotToken?: string
  telegramUserId?: string
  os?: 'windows'
}

export function buildSetupQuery(params: SetupQueryParams, maskKey = false): string {
  const q = new URLSearchParams()
  q.set('tool', params.tool)
  const key = params.key || 'your-api-key'
  q.set('key', maskKey ? maskApiKey(key) : key)
  q.set('serverUrl', params.serverUrl.replace(/\/v1\/?$/, ''))
  if (params.provider) q.set('provider', params.provider)
  if (params.model) q.set('model', params.model)
  if (params.models && params.models.length > 0) q.set('models', params.models.join(','))
  if (params.subagentModel) q.set('subagentModel', params.subagentModel)
  if (params.haiku) q.set('haiku', params.haiku)
  if (params.sonnet) q.set('sonnet', params.sonnet)
  if (params.opus) q.set('opus', params.opus)
  if (params.telegramBotToken) q.set('telegramBotToken', params.telegramBotToken)
  if (params.telegramUserId) q.set('telegramUserId', params.telegramUserId)
  if (params.os === 'windows') q.set('os', 'windows')
  return q.toString()
}

export function maskApiKey(key: string): string {
  if (!key || key === 'your-api-key') return 'your-api-key'
  if (key.length <= 12) return '***'
  return `${key.slice(0, 8)}...${key.slice(-4)}`
}

export function setupEndpointBase(origin: string): string {
  return `${origin.replace(/\/$/, '')}/api/v1/public/llm-setup`
}

export function buildSetupCommands(origin: string, params: SetupQueryParams) {
  const base = setupEndpointBase(origin)
  const displayUnix = `curl -sL "${base}?${buildSetupQuery(params, true)}" | bash`
  const displayWin = `irm "${base}?${buildSetupQuery({ ...params, os: 'windows' }, true)}" | iex`
  const realUnix = `curl -sL "${base}?${buildSetupQuery(params)}" | bash`
  const realWin = `irm "${base}?${buildSetupQuery({ ...params, os: 'windows' })}" | iex`
  return { displayUnix, displayWin, realUnix, realWin }
}