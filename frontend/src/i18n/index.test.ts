import {afterEach, describe, expect, it, vi} from 'vitest'

import {getBrowserLanguage, SUPPORTED_LOCALES} from './index'

describe('getBrowserLanguage', () => {
	afterEach(() => {
		vi.unstubAllGlobals()
	})

	it('picks pt-BR when the browser only reports pt', () => {
		vi.stubGlobal('navigator', {language: 'pt'})

		expect(getBrowserLanguage()).toBe('pt-BR')
	})

	it('keeps pt-PT when the browser reports it', () => {
		vi.stubGlobal('navigator', {language: 'pt-PT'})

		expect(getBrowserLanguage()).toBe('pt-PT')
	})

	it('keeps pt-BR when the browser reports it', () => {
		vi.stubGlobal('navigator', {language: 'pt-BR'})

		expect(getBrowserLanguage()).toBe('pt-BR')
	})

	it('still matches other bare language codes to their first regional locale', () => {
		vi.stubGlobal('navigator', {language: 'de'})

		expect(getBrowserLanguage()).toBe('de-DE')
	})

	it('falls back to english for an unsupported language', () => {
		vi.stubGlobal('navigator', {language: 'xx'})

		expect(getBrowserLanguage()).toBe('en')
	})
})

describe('SUPPORTED_LOCALES', () => {
	it('names both portuguese variants by country', () => {
		expect(SUPPORTED_LOCALES['pt-BR']).toBe('Português (Brasil)')
		expect(SUPPORTED_LOCALES['pt-PT']).toBe('Português (Portugal)')
	})
})
