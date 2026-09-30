import { useCallback, useState, type ReactNode } from 'react'
import SaveToast from './SaveToast'

/** Shared centered “Changes saved” dialog for Settings pages. */
export function useSaveToast(defaultMessage = 'Changes saved'): {
  toast: ReactNode
  showSaved: (message?: string) => void
} {
  const [open, setOpen] = useState(false)
  const [message, setMessage] = useState(defaultMessage)

  const showSaved = useCallback((msg = defaultMessage) => {
    setMessage(msg)
    setOpen(true)
  }, [defaultMessage])

  const dismiss = useCallback(() => setOpen(false), [])

  return {
    toast: (
      <SaveToast
        open={open}
        message={message}
        detail="Your changes have been saved successfully."
        onClose={dismiss}
      />
    ),
    showSaved,
  }
}
