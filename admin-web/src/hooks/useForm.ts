import { useState } from 'react'
import type { ChangeEvent } from 'react'

export function useForm<T extends Record<string, string>>(initial: T) {
  const [values, setValues] = useState<T>(initial)
  const setField = (key: keyof T) => (event: ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    setValues((current) => ({ ...current, [key]: event.target.value }))
  }
  const reset = () => setValues({ ...initial })
  return { values, setValues, setField, reset }
}
