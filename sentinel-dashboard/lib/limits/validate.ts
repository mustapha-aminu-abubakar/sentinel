type ValidationResult =
  | { ok: true }
  | { ok: false; errors: Record<string, string> }

const MAX_API_LEN = 255
const MAX_NAME_LEN = 255
const MAX_REQUESTS = 10_000_000
const MAX_WINDOW_SECONDS = 86_400 // 24 hours

export function validateRuleInput(input: {
  clientId: string
  api: string
  requestsAllowed: number
  windowSeconds: number
}): ValidationResult {
  const errors: Record<string, string> = {}

  if (!input.clientId.trim()) {
    errors.clientId = 'Client is required'
  }

  const api = input.api.trim()
  if (!api) {
    errors.api = 'API identifier is required'
  } else if (api.length > MAX_API_LEN) {
    errors.api = `API identifier must be ${MAX_API_LEN} characters or fewer`
  }

  if (
    !Number.isFinite(input.requestsAllowed) ||
    !Number.isInteger(input.requestsAllowed) ||
    input.requestsAllowed <= 0
  ) {
    errors.requestsAllowed = 'Must be a whole number greater than 0'
  } else if (input.requestsAllowed > MAX_REQUESTS) {
    errors.requestsAllowed = `Must be ${MAX_REQUESTS.toLocaleString()} or fewer`
  }

  if (
    !Number.isFinite(input.windowSeconds) ||
    !Number.isInteger(input.windowSeconds) ||
    input.windowSeconds <= 0
  ) {
    errors.windowSeconds = 'Must be a whole number greater than 0'
  } else if (input.windowSeconds > MAX_WINDOW_SECONDS) {
    errors.windowSeconds = `Must be ${MAX_WINDOW_SECONDS.toLocaleString()} seconds (24 h) or fewer`
  }

  return Object.keys(errors).length > 0 ? { ok: false, errors } : { ok: true }
}

export function validateClientInput(input: { name: string }): ValidationResult {
  const errors: Record<string, string> = {}

  const name = input.name.trim()
  if (!name) {
    errors.name = 'Client name is required'
  } else if (name.length > MAX_NAME_LEN) {
    errors.name = `Client name must be ${MAX_NAME_LEN} characters or fewer`
  }

  return Object.keys(errors).length > 0 ? { ok: false, errors } : { ok: true }
}
