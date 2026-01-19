import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { userApi } from '../api/client'

export default function CallbackPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { setAuth } = useAuthStore()

  useEffect(() => {
    const token = searchParams.get('token')

    if (token) {
      // Temporarily set token to make API call
      useAuthStore.setState({ accessToken: token })

      // Fetch user info
      userApi.me()
        .then((user) => {
          setAuth(user, token)
          navigate('/')
        })
        .catch((error) => {
          console.error('Failed to fetch user:', error)
          navigate('/login')
        })
    } else {
      navigate('/login')
    }
  }, [searchParams, navigate, setAuth])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
        <p className="mt-4 text-gray-600">Completing sign in...</p>
      </div>
    </div>
  )
}
