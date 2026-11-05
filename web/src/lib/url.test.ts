import { describe, expect, it } from 'vitest'
import { hostname, normalizeUrl } from './url'

describe('normalizeUrl', () => {
  it.each([
    ['google.com/careers', 'https://google.com/careers'],
    ['  google.com  ', 'https://google.com'],
    ['//google.com', 'https://google.com'],
    ['http://google.com', 'http://google.com'],
    ['', ''],
  ])('%j -> %j', (input, want) => expect(normalizeUrl(input)).toBe(want))
})

describe('hostname', () => {
  it('strips scheme, path and www', () => {
    expect(hostname('https://www.google.com/careers/x')).toBe('google.com')
    expect(hostname('google.com/careers')).toBe('google.com')
  })
})
