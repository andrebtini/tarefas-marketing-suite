<template>
	<div
		class="card"
		:class="{'has-no-shadow': !shadow}"
	>
		<header
			v-if="title !== ''"
			class="card-header"
		>
			<p class="card-header-title">
				{{ title }}
			</p>
			<BaseButton
				v-if="showClose"
				class="card-header-icon close"
				:aria-label="$t('misc.close')"
				@click="$emit('close')"
			>	
				<span class="icon">
					<Icon icon="times" />
				</span>
			</BaseButton>
		</header>
		<div
			class="card-content loader-container"
			:class="{
				'p-0': !padding,
				'is-loading': loading
			}"
		>
			<div :class="{'content': hasContent}">
				<slot />
			</div>
		</div>

		<footer
			v-if="$slots.footer"
			class="card-footer"
		>
			<slot name="footer" />
		</footer>
	</div>
</template>

<script setup lang="ts">
import BaseButton from '@/components/base/BaseButton.vue'

withDefaults(defineProps<{
	title?: string
	padding?: boolean
	shadow?: boolean
	hasContent?: boolean
	loading?: boolean
	showClose?: boolean
}>(), {
	title: '',
	padding: true,
	shadow: true,
	hasContent: true,
	loading: false,
	showClose: false,
})

defineEmits<{
	'close': []
}>()
</script>

<style lang="scss" scoped>
.card {
	background-color: var(--white);
	border-radius: $radius-large;
	margin-block-end: 1rem;
	// Flat card: the 1px border separates it from the board, no resting shadow.
	// The shadow prop and .has-no-shadow stay for API compatibility.
	border: 1px solid var(--card-border-color);
	color: var(--text);
	max-inline-size: 100%;
	position: relative;

	@media print {
		box-shadow: none;
		border: none;
	}
}

.card-header {
	background-color: transparent;
	align-items: stretch;
	display: flex;
	box-shadow: none;
	border-inline-end: 1px solid var(--card-border-color);
	border-radius: $radius-large $radius-large 0 0;
}

.card-header-title {
	align-items: center;
	color: var(--text-strong);
	display: flex;
	flex-grow: 1;
	font-weight: 600;
	padding: 0.75rem 1rem;

	&.is-centered {
		justify-content: center;
	}
}

.card-header-icon {
	align-items: center;
	cursor: pointer;
	display: flex;
	justify-content: center;
	padding: 0.75rem 1rem;
}

.card-content {
	background-color: transparent;
	padding: 1.5rem;

	&:first-child {
		border-start-start-radius: $radius-large;
		border-start-end-radius: $radius-large;
	}

	&:last-child {
		border-end-start-radius: $radius-large;
		border-end-end-radius: $radius-large;
	}

	// Utility classes like .p-0 are defined globally with lower specificity
	// than Vue-scoped selectors; restore precedence explicitly.
	&.p-0 {
		padding: 0;
	}
}

.card-footer {
	align-items: stretch;
	background-color: var(--grey-50);
	border-block-start: 1px solid var(--card-border-color);
	// Inner radius of a 1px bordered card, so the tinted footer does not poke out of the corners
	border-end-start-radius: $radius-large - 1px;
	border-end-end-radius: $radius-large - 1px;
	padding: 20px;
	display: flex;
	justify-content: flex-end;
}
</style>
