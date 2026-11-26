import { useCallback, useEffect, useState } from 'react'
import type { DependencyList } from 'react'

export interface ResourceState<T> {
  data: T | null
  loading: boolean
  error: string | null
  reload: () => void
}

export function useResource<T>(loader: () => Promise<T>, deps: DependencyList): ResourceState<T> {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [version, setVersion] = useState(0)
  const reload = useCallback(() => setVersion((value) => value + 1), [])

  useEffect(() => {
    let active = true
    setLoading(true)
    setError(null)
    loader()
      .then((result) => {
        if (active) setData(result)
      })
      .catch((reason: unknown) => {
        if (active) {
          setData(null)
          setError(reason instanceof Error ? reason.message : 'Request failed')
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [...deps, version])

  return { data, loading, error, reload }
}
