import React, { createContext, useContext, useState } from 'react'

export type Language = 'en' | 'vi'

const translations = {
  en: {
    // sidebar
    'sidebar.overview': 'Overview',
    'sidebar.dashboard': 'Dashboard',
    'sidebar.playground': 'Playground',
    'sidebar.manage': 'Manage',
    'sidebar.keys': 'Keys',
    'sidebar.teams': 'Teams',
    'sidebar.service_accounts': 'Service Accounts',
    'sidebar.mcp_servers': 'MCP Servers',
    'sidebar.analytics': 'Analytics',
    'sidebar.usage': 'Usage',
    'sidebar.cost_reports': 'Cost Reports',
    'sidebar.security': 'Security',
    'sidebar.audit_log': 'Audit Log',
    'sidebar.sso_config': 'SSO Config',
    'sidebar.organization': 'Organization',
    'sidebar.system': 'System',
    'sidebar.organizations': 'Organizations',
    'sidebar.users': 'Users',
    'sidebar.models': 'Models',
    'sidebar.license': 'License',
    'sidebar.profile': 'Profile',
    'sidebar.logout': 'Logout',
    // login
    'login.title': 'VoidLLM',
    'login.subtitle': 'Sign in to your workspace',
    'login.email': 'Email',
    'login.password': 'Password',
    'login.sign_in': 'Sign in',
    'login.or': 'or',
    'login.sso': 'Sign in with SSO',
    // dashboard
    'dashboard.requests': 'Requests',
    'dashboard.requests_desc': 'Total requests in last 24h',
    'dashboard.tokens': 'Tokens',
    'dashboard.tokens_desc': 'Token consumption in last 24h',
    'dashboard.cost': 'Est. Cost',
    'dashboard.cost_desc': 'Total estimated cost in last 24h',
    'dashboard.keys': 'Active Keys',
    'dashboard.keys_desc': 'Active API keys in last 24h',
    'dashboard.throughput': 'Request Throughput',
    'dashboard.throughput_desc': 'Requests per second',
    'dashboard.latency': 'Response Latency',
    'dashboard.latency_desc': 'Average duration in milliseconds',
    'dashboard.token_count': 'Token Consumption',
    'dashboard.token_count_desc': 'Input vs Output tokens',
    'dashboard.model_distribution': 'Model Distribution',
    'dashboard.model_distribution_desc': 'Percentage of requests by model',
    'dashboard.active_streams': 'Active Streams',
    'dashboard.active_streams_desc': 'Real-time server-sent events',
    'dashboard.overview_title': 'Overview',
    'dashboard.overview_desc': 'System performance metrics',
    // playground
    'playground.title': 'Playground',
    'playground.desc': 'Test model endpoints and tools',
    'playground.select_model': 'Select a model...',
    'playground.system_prompt': 'System Prompt',
    'playground.system_prompt_placeholder': 'Instructions for the model...',
    'playground.temp': 'Temperature',
    'playground.max_tokens': 'Max Tokens',
    'playground.stream': 'Stream response',
    'playground.type_message': 'Type a message...',
    'playground.send': 'Send',
    'playground.clear': 'Clear Chat',
    'playground.send_hint': 'Enter to send · Shift+Enter for new line',
    'playground.no_models': 'No models configured. Go to Models page to add one.',
    // keys
    'keys.title': 'API Keys',
    'keys.desc': 'Manage access keys for your applications',
    'keys.add': 'Create API Key',
    'keys.table.name': 'Name',
    'keys.table.hint': 'Key Hint',
    'keys.table.type': 'Type',
    'keys.table.limits': 'Limits',
    'keys.table.expires': 'Expires',
    'keys.table.last_used': 'Last Used',
    'keys.table.actions': 'Actions',
    'keys.type.user': 'User',
    'keys.type.team': 'Team',
    'keys.type.sa': 'Service Account',
    'keys.limit.none': 'None',
    'keys.limit.daily_tokens': '{{limit}} tokens/day',
    'keys.limit.monthly_tokens': '{{limit}} tokens/month',
    'keys.limit.rpm': '{{limit}} RPM',
    'keys.limit.rpd': '{{limit}} RPD',
    // license
    'license.title': 'License',
    'license.desc': 'Manage your VoidLLM license',
    'license.current_plan': 'Current Plan',
    'license.never': 'Never',
    'license.max_orgs': 'Max Orgs',
    'license.max_teams': 'Max Teams',
    'license.customer_id': 'Customer ID',
    'license.key': 'License Key',
    'license.key_replace': 'Replace License Key',
    'license.key_desc_community': 'Paste your license key to activate Pro or Enterprise features.',
    'license.key_desc_paid': 'Paste a new license key to change your plan.',
    'license.activate': 'Activate License',
    'license.restart_notice': 'License saved. Restart VoidLLM to activate.',
    'license.fail_notice': 'Failed to activate license',
    'license.save_changes': 'Save Changes',
    'license.pro.title': 'Pro',
    'license.pro.desc': '$2,990/yr (save 2 months)',
    'license.pro.upgrade': 'Upgrade to Pro',
    'license.ent.title': 'Enterprise',
    'license.ent.desc_community': '$7,990/yr (save 2 months) · Everything in Pro, plus:',
    'license.ent.desc_pro': '$7,990/yr (save 2 months) · Everything in your current plan, plus:',
    'license.ent.contact': 'Contact Sales',
  },
  vi: {
    // sidebar
    'sidebar.overview': 'Tổng quan',
    'sidebar.dashboard': 'Bảng điều khiển',
    'sidebar.playground': 'Playground',
    'sidebar.manage': 'Quản lý',
    'sidebar.keys': 'Khóa API',
    'sidebar.teams': 'Nhóm',
    'sidebar.service_accounts': 'Tài khoản dịch vụ',
    'sidebar.mcp_servers': 'Máy chủ MCP',
    'sidebar.analytics': 'Phân tích',
    'sidebar.usage': 'Lượng sử dụng',
    'sidebar.cost_reports': 'Báo cáo chi phí',
    'sidebar.security': 'Bảo mật',
    'sidebar.audit_log': 'Nhật ký kiểm toán',
    'sidebar.sso_config': 'Cấu hình SSO',
    'sidebar.organization': 'Tổ chức',
    'sidebar.system': 'Hệ thống',
    'sidebar.organizations': 'Các tổ chức',
    'sidebar.users': 'Người dùng',
    'sidebar.models': 'Mô hình AI',
    'sidebar.license': 'Bản quyền',
    'sidebar.profile': 'Hồ sơ',
    'sidebar.logout': 'Đăng xuất',
    // login
    'login.title': 'VoidLLM',
    'login.subtitle': 'Đăng nhập vào không gian làm việc',
    'login.email': 'Email',
    'login.password': 'Mật khẩu',
    'login.sign_in': 'Đăng nhập',
    'login.or': 'hoặc',
    'login.sso': 'Đăng nhập với SSO',
    // dashboard
    'dashboard.requests': 'Yêu cầu (Requests)',
    'dashboard.requests_desc': 'Tổng số yêu cầu trong 24h qua',
    'dashboard.tokens': 'Tokens',
    'dashboard.tokens_desc': 'Lượng token tiêu thụ trong 24h qua',
    'dashboard.cost': 'Chi phí ước tính',
    'dashboard.cost_desc': 'Tổng chi phí ước tính trong 24h qua',
    'dashboard.keys': 'Khóa hoạt động',
    'dashboard.keys_desc': 'Khóa API hoạt động trong 24h qua',
    'dashboard.throughput': 'Thông lượng yêu cầu',
    'dashboard.throughput_desc': 'Yêu cầu mỗi giây',
    'dashboard.latency': 'Độ trễ phản hồi',
    'dashboard.latency_desc': 'Thời gian phản hồi trung bình (ms)',
    'dashboard.token_count': 'Tiêu thụ Token',
    'dashboard.token_count_desc': 'Token Input so với Output',
    'dashboard.model_distribution': 'Phân bổ mô hình',
    'dashboard.model_distribution_desc': 'Tỷ lệ yêu cầu theo mô hình',
    'dashboard.active_streams': 'Luồng hoạt động',
    'dashboard.active_streams_desc': 'Thời gian thực (Server-sent events)',
    'dashboard.overview_title': 'Tổng quan',
    'dashboard.overview_desc': 'Chỉ số hiệu suất hệ thống',
    // playground
    'playground.title': 'Playground',
    'playground.desc': 'Kiểm tra hoạt động các mô hình và công cụ',
    'playground.select_model': 'Chọn một mô hình...',
    'playground.system_prompt': 'Chỉ chỉ thị hệ thống (System Prompt)',
    'playground.system_prompt_placeholder': 'Hướng dẫn hoạt động cho mô hình...',
    'playground.temp': 'Nhiệt độ (Temperature)',
    'playground.max_tokens': 'Tokens tối đa',
    'playground.stream': 'Phản hồi dạng luồng (Stream)',
    'playground.type_message': 'Nhập tin nhắn...',
    'playground.send': 'Gửi',
    'playground.clear': 'Xóa đoạn chat',
    'playground.send_hint': 'Enter để gửi · Shift+Enter để xuống dòng',
    'playground.no_models': 'Chưa cấu hình mô hình nào. Hãy vào mục Mô hình để thêm.',
    // keys
    'keys.title': 'Khóa API',
    'keys.desc': 'Quản lý khóa truy cập ứng dụng của bạn',
    'keys.add': 'Tạo API Key',
    'keys.table.name': 'Tên khóa',
    'keys.table.hint': 'Gợi ý khóa',
    'keys.table.type': 'Loại',
    'keys.table.limits': 'Giới hạn',
    'keys.table.expires': 'Hết hạn',
    'keys.table.last_used': 'Dùng lần cuối',
    'keys.table.actions': 'Thao tác',
    'keys.type.user': 'Người dùng',
    'keys.type.team': 'Nhóm',
    'keys.type.sa': 'Tài khoản dịch vụ',
    'keys.limit.none': 'Không giới hạn',
    'keys.limit.daily_tokens': '{{limit}} tokens/ngày',
    'keys.limit.monthly_tokens': '{{limit}} tokens/tháng',
    'keys.limit.rpm': '{{limit}} RPM (Yêu cầu/phút)',
    'keys.limit.rpd': '{{limit}} RPD (Yêu cầu/ngày)',
    // license
    'license.title': 'Bản quyền',
    'license.desc': 'Quản lý bản quyền VoidLLM của bạn',
    'license.current_plan': 'Gói hiện tại',
    'license.never': 'Không bao giờ',
    'license.max_orgs': 'Số Tổ chức tối đa',
    'license.max_teams': 'Số Nhóm tối đa',
    'license.customer_id': 'ID Khách hàng',
    'license.key': 'Mã bản quyền (License Key)',
    'license.key_replace': 'Thay thế mã bản quyền',
    'license.key_desc_community': 'Dán mã bản quyền để kích hoạt các tính năng Pro hoặc Enterprise.',
    'license.key_desc_paid': 'Dán mã bản quyền mới để thay đổi gói dịch vụ.',
    'license.activate': 'Kích hoạt bản quyền',
    'license.restart_notice': 'Đã lưu bản quyền. Vui lòng khởi động lại VoidLLM để kích hoạt.',
    'license.fail_notice': 'Kích hoạt bản quyền thất bại',
    'license.save_changes': 'Lưu thay đổi',
    'license.pro.title': 'Pro',
    'license.pro.desc': '$2.990/năm (tiết kiệm 2 tháng)',
    'license.pro.upgrade': 'Nâng cấp lên Pro',
    'license.ent.title': 'Enterprise',
    'license.ent.desc_community': '$7.990/năm (tiết kiệm 2 tháng) · Bao gồm gói Pro và thêm:',
    'license.ent.desc_pro': '$7.990/năm (tiết kiệm 2 tháng) · Bao gồm gói hiện tại của bạn và thêm:',
    'license.ent.contact': 'Liên hệ bộ phận bán hàng',
  }
}

export type TranslationKey = keyof typeof translations.en

interface TranslationContextType {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: TranslationKey, variables?: Record<string, string | number>) => string
}

const TranslationContext = createContext<TranslationContextType | undefined>(undefined)

export function TranslationProvider({ children }: { children: React.ReactNode }) {
  // Set default language to Vietnamese ('vi') as requested
  const [language, setLanguageState] = useState<Language>(() => {
    const saved = localStorage.getItem('voidllm_lang')
    return (saved === 'en' || saved === 'vi') ? saved : 'vi'
  })

  const setLanguage = (lang: Language) => {
    setLanguageState(lang)
    localStorage.setItem('voidllm_lang', lang)
  }

  const t = (key: TranslationKey, variables?: Record<string, string | number>): string => {
    const langTrans = translations[language] || translations['en']
    let text = langTrans[key] || translations['en'][key] || String(key)

    if (variables) {
      Object.entries(variables).forEach(([k, v]) => {
        text = text.replace(new RegExp(`{{${k}}}`, 'g'), String(v))
      })
    }
    return text
  }

  return (
    <TranslationContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </TranslationContext.Provider>
  )
}

export function useTranslation() {
  const context = useContext(TranslationContext)
  if (!context) {
    throw new Error('useTranslation must be used within a TranslationProvider')
  }
  return context
}
