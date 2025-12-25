import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { LoginRequest, User } from '@/types'

export function useAuth() {
  const queryClient = useQueryClient()

  // 現在のユーザー情報を取得
  const {
    data: user,
    isLoading,
    error,
  } = useQuery<User | null>({
    queryKey: ['auth', 'me'],
    queryFn: async () => api.auth.me(),
    retry: false,
    staleTime: 0, // 常に再検証する（開発用）
    refetchOnMount: 'always',
    refetchOnWindowFocus: false,
  })

  // ログイン mutation
  const loginMutation = useMutation({
    mutationFn: (data: LoginRequest) => api.auth.login(data),
    onSuccess: (res) => {
      // ログイン直後にキャッシュを即座に更新し、念のため再取得もかける
      queryClient.setQueryData(['auth', 'me'], res.user)
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] })
    },
  })

  // ログアウト mutation
  const logoutMutation = useMutation({
    mutationFn: () => api.auth.logout(),
    onSuccess: () => {
      // ログアウト後、ユーザー情報をクリア
      queryClient.setQueryData(['auth', 'me'], null)
      queryClient.clear() // 全てのキャッシュをクリア
    },
  })

  return {
    user,
    isLoading,
    error,
    login: loginMutation.mutateAsync,
    logout: logoutMutation.mutateAsync,
    isAuthenticated: !!user,
    isAdmin: user?.role === 'admin',
    isLoginLoading: loginMutation.isPending,
    isLogoutLoading: logoutMutation.isPending,
  }
}
