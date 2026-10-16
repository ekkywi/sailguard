export type AgentChannel = {
  id: string
  label: string
  description: string
  os_family: string
  toolchain: string
  binary_name: string
  binary_path: string
  binary_available: boolean
  binary_size_bytes: number
  binary_modified_at: string
  build_hint: string
  downloadable: boolean
  coming_soon: boolean
}

export type AgentPackageMeta = {
  default_server_url: string
  default_data_dir?: string
  setup_binary_name?: string
  setup_binary_path?: string
  setup_binary_available?: boolean
  setup_binary_size_bytes?: number
  setup_binary_modified_at?: string
  setup_build_hint?: string
  channels: AgentChannel[]
}

export type AgentPackageKind = 'install' | 'update'

export type DownloadWindowsPackageInput = {
  channel: string
  package_kind?: AgentPackageKind
  server_url: string
  enrollment_token?: string
  data_dir?: string
}
