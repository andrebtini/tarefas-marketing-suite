import {afterEach, describe, expect, it, vi} from 'vitest'
import {effectScope} from 'vue'
import dayjs from 'dayjs'

import {i18n} from '@/i18n'
import {
	DAYJS_LANGUAGE_IMPORTS,
	DAYJS_LOCALE_MAPPING,
	useDayjsLanguageSync,
} from './useDayjsLanguageSync'

describe('useDayjsLanguageSync', () => {
	afterEach(() => {
		i18n.global.locale.value = 'en'
		dayjs.locale('en')
	})

	it('switches the global dayjs locale to pt-br for pt-BR', async () => {
		i18n.global.locale.value = 'pt-BR'
		const scope = effectScope()
		const isLoading = scope.run(() => useDayjsLanguageSync(dayjs))

		await vi.waitFor(() => expect(isLoading?.value).toBe(false))
		scope.stop()

		expect(dayjs.locale()).toBe('pt-br')
		expect(dayjs(new Date(2026, 8, 21, 9, 0)).format('dddd, D [de] MMMM')).toBe('segunda-feira, 21 de setembro')
		expect(dayjs(new Date(2026, 8, 21, 9, 0)).from(new Date(2026, 8, 21, 9, 5))).toBe('há 5 minutos')
	})
})

describe('DAYJS_LOCALE_MAPPING', () => {
	it('points every language to the dayjs locale it imports', async () => {
		// The mapping is keyed by the lowercased language code, not by SupportedLocale
		const mapping = DAYJS_LOCALE_MAPPING as Record<string, string>

		for (const [language, importLocale] of Object.entries(DAYJS_LANGUAGE_IMPORTS)) {
			const {default: locale} = await importLocale() as unknown as {default: {name: string}}
			expect(mapping[language], language).toBe(locale.name)
		}
	})
})
