import { createContext, ReactNode, useContext } from 'react'

export type UserRole = 'admin' | 'viewer'

export interface AuthUser {
  id: string
  username: string
  role: UserRole
  email?: string
  tenant_id?: string
}

export function roleLabel(role: UserRole | string) {
  return role === 'admin' ? 'Admin' : 'User'
}

interface AuthContextValue {
  user: AuthUser | null
  isAdmin: boolean
  isPlatformAdmin: boolean
  isCustomerAdmin: boolean
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue>({
  user: null,
  isAdmin: false,
  isPlatformAdmin: false,
  isCustomerAdmin: false,
  refresh: async () => {},
})

export function useAuth() {
  return useContext(AuthContext)
}

export function AuthProvider({
  children,
  user,
  refresh,
}: {
  children: ReactNode
  user: AuthUser | null
  refresh: () => Promise<void>
}) {
  const isAdmin = user?.role === 'admin'
  const isPlatformAdmin = !!isAdmin && !user?.tenant_id
  const isCustomerAdmin = !!isAdmin && !!user?.tenant_id
  return (
    <AuthContext.Provider value={{ user, isAdmin, isPlatformAdmin, isCustomerAdmin, refresh }}>
      {children}
    </AuthContext.Provider>
  )
}
