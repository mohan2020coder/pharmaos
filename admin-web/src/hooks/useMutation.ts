import { useState } from 'react'

import { errorMessage } from '../lib/format'

export function useMutation() {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = async <T>(action: () => Promise<T>): Promise<T | undefined> => {
    setBusy(true)
    setError(null)
    try {
      return await action()
    } catch (reason) {
      setError(errorMessage(reason))
      return undefined
    } finally {
      setBusy(false)
    }
  }

  return { busy, error, run, setError }
}
