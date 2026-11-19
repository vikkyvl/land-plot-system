export const homePath = '/'

const authPaths = ['/login', '/auth/callback']

export function safeReturnPath(path: string | null | undefined): string {
  if (!path || !path.startsWith('/') || path.startsWith('//') || path.includes('\\')) {
    return homePath
  }

  const pathname = path.split(/[?#]/, 1)[0] ?? ''
  if (authPaths.includes(pathname)) {
    return homePath
  }

  return path
}
