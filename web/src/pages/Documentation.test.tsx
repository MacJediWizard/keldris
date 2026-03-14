import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MarkdownRenderer } from './Documentation';

describe('MarkdownRenderer', () => {
	function renderMarkdown(content: string) {
		const { container } = render(<MarkdownRenderer content={content} />);
		return container.innerHTML;
	}

	function getRenderedDOM(content: string) {
		const { container } = render(<MarkdownRenderer content={content} />);
		return container;
	}

	describe('XSS prevention', () => {
		it('strips javascript: URIs from markdown links', () => {
			const html = renderMarkdown('[Click me](javascript:alert(1))');
			expect(html).not.toContain('javascript:');
			// The link text should still render
			expect(html).toContain('Click me');
		});

		it('strips javascript: URIs with mixed case', () => {
			const html = renderMarkdown('[Click](jAvAsCrIpT:alert(1))');
			expect(html).not.toContain('javascript:');
			expect(html).not.toContain('jAvAsCrIpT:');
		});

		it('strips data: URIs from markdown links', () => {
			renderMarkdown('[Click](data:text/html,<script>alert(1)</script>)');
			const container = getRenderedDOM(
				'[Click](data:text/html,<script>alert(1)</script>)',
			);
			// No anchor should have a data: href
			const anchors = container.querySelectorAll('a[href]');
			for (const a of anchors) {
				expect(a.getAttribute('href')).not.toMatch(/^data:/i);
			}
		});

		it('does not render img tags with onerror as executable HTML', () => {
			const container = getRenderedDOM('<img src=x onerror="alert(1)">');
			// No actual img element should exist in the DOM
			const imgs = container.querySelectorAll('img');
			expect(imgs.length).toBe(0);
		});

		it('does not render script tags as executable HTML', () => {
			const container = getRenderedDOM('<script>alert(1)</script>');
			// No actual script element should exist in the DOM
			const scripts = container.querySelectorAll('script');
			expect(scripts.length).toBe(0);
		});

		it('does not render elements with event handlers as executable HTML', () => {
			const container = getRenderedDOM(
				'<div onmouseover="alert(1)">hover me</div>',
			);
			// No element should have an onmouseover attribute
			const elements = container.querySelectorAll('[onmouseover]');
			expect(elements.length).toBe(0);
		});

		it('strips vbscript: URIs from markdown links', () => {
			renderMarkdown('[Click](vbscript:MsgBox(1))');
			const container = getRenderedDOM('[Click](vbscript:MsgBox(1))');
			const anchors = container.querySelectorAll('a[href]');
			for (const a of anchors) {
				expect(a.getAttribute('href')).not.toMatch(/^vbscript:/i);
			}
		});
	});

	describe('normal markdown rendering', () => {
		it('renders h1 headers', () => {
			const html = renderMarkdown('# Title');
			expect(html).toContain('<h1');
			expect(html).toContain('Title');
		});

		it('renders h2 headers', () => {
			const html = renderMarkdown('## Subtitle');
			expect(html).toContain('<h2');
			expect(html).toContain('Subtitle');
		});

		it('renders h3 headers', () => {
			const html = renderMarkdown('### Section');
			expect(html).toContain('<h3');
			expect(html).toContain('Section');
		});

		it('renders bold text', () => {
			const html = renderMarkdown('**bold text**');
			expect(html).toContain('<strong>bold text</strong>');
		});

		it('renders italic text', () => {
			const html = renderMarkdown('*italic text*');
			expect(html).toContain('<em>italic text</em>');
		});

		it('renders safe links with correct href', () => {
			const container = getRenderedDOM('[Keldris](https://example.com)');
			const anchor = container.querySelector('a');
			expect(anchor).not.toBeNull();
			expect(anchor?.getAttribute('href')).toBe('https://example.com');
			expect(anchor?.textContent).toBe('Keldris');
		});

		it('renders inline code', () => {
			const html = renderMarkdown('`some code`');
			expect(html).toContain('<code');
			expect(html).toContain('some code');
		});

		it('renders list items', () => {
			const html = renderMarkdown('- item one');
			expect(html).toContain('<li');
			expect(html).toContain('item one');
		});
	});
});
