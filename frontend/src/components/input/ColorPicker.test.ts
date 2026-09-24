import {describe, it, expect} from 'vitest'
import {mount} from '@vue/test-utils'
import ColorPicker from './ColorPicker.vue'

function suggestedColors(): string[] {
	const wrapper = mount(ColorPicker, {
		props: {modelValue: ''},
		global: {
			mocks: {$t: (key: string) => key},
		},
	})
	return wrapper.findAll('datalist option').map(option => (option.element as HTMLOptionElement).value)
}

describe('ColorPicker', () => {
	it('suggests the brand primary as the first color', () => {
		expect(suggestedColors()[0]).toBe('#0042bc')
	})

	it('keeps six suggestions and drops the old default primary', () => {
		const colors = suggestedColors()
		expect(colors).toHaveLength(6)
		expect(colors).not.toContain('#1973ff')
	})
})
