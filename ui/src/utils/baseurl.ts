export function apiBaseURL() {
  const { protocol, hostname, host } = window.location
  if (import.meta.env.DEV) {
    return `${protocol}//${hostname}:1323`
  }
  return `${protocol}//${host}`
}

export function apiWsURL() {
  const { protocol, hostname, host } = window.location
  const wsProto = protocol === 'https:' ? 'wss:' : 'ws:'
  if (import.meta.env.DEV) {
    return `${wsProto}//${hostname}:1323`
  }
  return `${wsProto}//${host}`
}
