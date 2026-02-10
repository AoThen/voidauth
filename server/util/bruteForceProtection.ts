import { logger } from './logger'
import appConfig from './config'

export interface LoginAttempt {
  count: number
  firstAttempt: Date
  lastAttempt: Date
  blockedUntil?: Date
}

export class LoginBruteForceProtection {
  private attempts: Map<string, LoginAttempt> = new Map()

  constructor(
    private maxAttempts: number = appConfig.LOGIN_MAX_ATTEMPTS,
    private blockDurationMinutes: number = appConfig.LOGIN_BLOCK_DURATION,
  ) {}

  getClientIP(
    req: {
      headers: {
        'x-forwarded-for'?: string | string[]
        'cf-connecting-ip'?: string | string[]
        'x-real-ip'?: string | string[]
      }
      socket?: { remoteAddress?: string }
    },
  ): string {
    const forwarded = req.headers['x-forwarded-for']
    if (forwarded) {
      const ips = Array.isArray(forwarded) ? forwarded[0] : forwarded.split(',')[0]
      if (ips) {
        return ips.trim() || 'unknown'
      }
    }
    const cfIP = req.headers['cf-connecting-ip']
    if (cfIP) {
      const ip = Array.isArray(cfIP) ? cfIP[0] : cfIP
      return ip || 'unknown'
    }
    const realIP = req.headers['x-real-ip']
    if (realIP) {
      const ip = Array.isArray(realIP) ? realIP[0] : realIP
      return ip || 'unknown'
    }
    return req.socket?.remoteAddress?.replace('::ffff:', '') || 'unknown'
  }

  getIdentifier(input: string, ip: string): string {
    return `${input.toLowerCase()}:${ip}`
  }

  isBlocked(identifier: string): boolean {
    const attempt = this.attempts.get(identifier)
    if (!attempt) {
      return false
    }
    if (attempt.blockedUntil && new Date() < attempt.blockedUntil) {
      return true
    }
    if (attempt.blockedUntil && new Date() >= attempt.blockedUntil) {
      this.attempts.delete(identifier)
      return false
    }
    return false
  }

  getRemainingBlockTime(identifier: string): number {
    const attempt = this.attempts.get(identifier)
    if (!attempt?.blockedUntil) {
      return 0
    }
    const remaining = Math.ceil((attempt.blockedUntil.getTime() - Date.now()) / 1000 / 60)
    return Math.max(0, remaining)
  }

  recordFailedAttempt(
    input: string,
    ip: string,
  ): { blocked: boolean, remainingAttempts: number, blockTimeMinutes?: number } {
    const identifier = this.getIdentifier(input, ip)
    let attempt = this.attempts.get(identifier)
    const now = new Date()

    if (!attempt) {
      attempt = {
        count: 1,
        firstAttempt: now,
        lastAttempt: now,
      }
    } else {
      attempt.count++
      attempt.lastAttempt = now

      if (this.isBlocked(identifier)) {
        return { blocked: true, remainingAttempts: 0, blockTimeMinutes: this.blockDurationMinutes }
      }
    }

    if (attempt.count >= this.maxAttempts) {
      attempt.blockedUntil = new Date(now.getTime() + this.blockDurationMinutes * 60 * 1000)
      this.attempts.set(identifier, attempt)

      logger.info(
        `Login blocked for ${input} (IP: ${ip}) after ${String(attempt.count)} failed attempts. `
        + `Blocked until ${attempt.blockedUntil.toISOString()}`,
      )

      return {
        blocked: true,
        remainingAttempts: 0,
        blockTimeMinutes: this.blockDurationMinutes,
      }
    }

    this.attempts.set(identifier, attempt)

    const remainingAttempts = this.maxAttempts - attempt.count

    if (attempt.count >= this.maxAttempts - 3) {
      logger.info(
        `Failed login attempt ${String(attempt.count)}/${String(this.maxAttempts)} for ${input} (IP: ${ip})`,
      )
    }

    return {
      blocked: false,
      remainingAttempts,
    }
  }

  recordSuccessfulAttempt(input: string, ip: string): void {
    const identifier = this.getIdentifier(input, ip)
    const attempt = this.attempts.get(identifier)
    if (attempt) {
      logger.info(`Successful login for ${input} (IP: ${ip}), resetting failed attempt counter`)
      this.attempts.delete(identifier)
    }
  }

  cleanup(): number {
    const now = new Date()
    let cleaned = 0
    for (const [key, attempt] of this.attempts.entries()) {
      if (attempt.blockedUntil && now >= attempt.blockedUntil) {
        this.attempts.delete(key)
        cleaned++
      } else if (!attempt.blockedUntil) {
        const expirationTime = 60 * 60 * 1000
        if (now.getTime() - attempt.firstAttempt.getTime() > expirationTime) {
          this.attempts.delete(key)
          cleaned++
        }
      }
    }
    return cleaned
  }

  getStats(): { totalBlocked: number, currentlyBlocked: number } {
    let totalBlocked = 0
    let currentlyBlocked = 0
    const now = new Date()
    for (const attempt of this.attempts.values()) {
      if (attempt.blockedUntil) {
        totalBlocked++
        if (now < attempt.blockedUntil) {
          currentlyBlocked++
        }
      }
    }
    return { totalBlocked, currentlyBlocked }
  }
}

export const bruteForceProtection = new LoginBruteForceProtection()

setInterval(() => {
  const cleaned = bruteForceProtection.cleanup()
  if (cleaned > 0) {
    logger.debug(`Cleaned up ${String(cleaned)} expired login attempts`)
  }
}, 5 * 60 * 1000)
