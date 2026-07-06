import type { TranslationKey } from '../i18n'

export interface ModelSlot {
  key: string
  label: string
  default: string
}

export interface GuideStep {
  title: string
  desc?: string
  code?: string
  copyable?: boolean
}

export type IntegrationToolId =
  | 'claude'
  | 'codex'
  | 'cline'
  | 'kilo'
  | 'hermes'
  | 'openclaw'
  | 'opencode'
  | 'jcode'

export interface IntegrationTool {
  id: IntegrationToolId
  name: string
  logo: string
  descKey: TranslationKey
  configType: 'auto' | 'guide'
  installCmd?: string
  modelSlots?: ModelSlot[]
  multiModel?: boolean
  telegram?: boolean
  guideSteps?: GuideStep[]
  docsUrl?: string
}

export const INTEGRATION_TOOLS: IntegrationTool[] = [
  {
    id: 'claude',
    name: 'Claude Code',
    logo: '/providers/claude.png',
    descKey: 'integrations.tool.claude',
    configType: 'auto',
    installCmd: 'npm install -g @anthropic-ai/claude-code',
    modelSlots: [
      { key: 'haiku', label: 'Haiku', default: 'claude-haiku-4-5' },
      { key: 'sonnet', label: 'Sonnet', default: 'claude-sonnet-4-5' },
      { key: 'opus', label: 'Opus', default: 'claude-opus-4-5' },
    ],
  },
  {
    id: 'codex',
    name: 'OpenAI Codex CLI',
    logo: '/providers/codex.png',
    descKey: 'integrations.tool.codex',
    configType: 'auto',
    installCmd: 'npm install -g @openai/codex',
    modelSlots: [
      { key: 'model', label: 'Main', default: 'gpt-4o' },
      { key: 'subagentModel', label: 'Subagent', default: '' },
    ],
  },
  {
    id: 'cline',
    name: 'Cline',
    logo: '/providers/cline.png',
    descKey: 'integrations.tool.cline',
    configType: 'auto',
    modelSlots: [{ key: 'model', label: 'Model', default: 'gpt-4o' }],
  },
  {
    id: 'kilo',
    name: 'Kilo Code',
    logo: '/providers/kilocode.png',
    descKey: 'integrations.tool.kilo',
    configType: 'auto',
    multiModel: true,
    modelSlots: [{ key: 'model', label: 'Default', default: 'gpt-4o' }],
  },
  {
    id: 'hermes',
    name: 'Hermes Agent',
    logo: '/providers/hermes.png',
    descKey: 'integrations.tool.hermes',
    configType: 'auto',
    installCmd:
      'curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash',
    modelSlots: [{ key: 'model', label: 'Model', default: 'gpt-4o' }],
    telegram: true,
  },
  {
    id: 'openclaw',
    name: 'Open Claw',
    logo: '/providers/openclaw.png',
    descKey: 'integrations.tool.openclaw',
    configType: 'auto',
    modelSlots: [{ key: 'model', label: 'Model', default: 'gpt-4o' }],
    telegram: true,
  },
  {
    id: 'opencode',
    name: 'OpenCode',
    logo: '/providers/opencode.png',
    descKey: 'integrations.tool.opencode',
    configType: 'auto',
    installCmd: 'npm install -g opencode-ai',
    modelSlots: [{ key: 'model', label: 'Model', default: 'gpt-4o' }],
  },
  {
    id: 'jcode',
    name: 'jcode',
    logo: '/providers/jcode.png',
    descKey: 'integrations.tool.jcode',
    configType: 'auto',
    installCmd:
      'curl -fsSL https://raw.githubusercontent.com/1jehuang/jcode/master/scripts/install.sh | bash',
    modelSlots: [{ key: 'model', label: 'Model', default: 'gpt-4o' }],
  },
]

export const TELEGRAM_USERINFO_BOT = 'https://t.me/userinfobot'
export const TELEGRAM_BOTFATHER = 'https://t.me/BotFather'