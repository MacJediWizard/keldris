import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Stepper, VerticalStepper } from './Stepper';

const steps = [
	{ id: 'step1', label: 'Step 1', description: 'First step' },
	{ id: 'step2', label: 'Step 2', description: 'Second step' },
	{ id: 'step3', label: 'Step 3', description: 'Third step' },
];

describe('Stepper', () => {
	it('renders all steps', () => {
		render(
			<Stepper steps={steps} currentStep="step1" completedSteps={[]} />,
		);
		expect(screen.getByText('Step 1')).toBeInTheDocument();
		expect(screen.getByText('Step 2')).toBeInTheDocument();
		expect(screen.getByText('Step 3')).toBeInTheDocument();
	});

	it('displays step numbers for non-completed steps', () => {
		render(
			<Stepper steps={steps} currentStep="step1" completedSteps={[]} />,
		);
		expect(screen.getByText('1')).toBeInTheDocument();
		expect(screen.getByText('2')).toBeInTheDocument();
		expect(screen.getByText('3')).toBeInTheDocument();
	});

	it('shows checkmark for completed steps', () => {
		const { container } = render(
			<Stepper
				steps={steps}
				currentStep="step2"
				completedSteps={['step1']}
			/>,
		);
		const svgs = container.querySelectorAll('svg');
		expect(svgs.length).toBeGreaterThan(0);
	});

	it('calls onStepClick when a completed step is clicked', () => {
		const onStepClick = vi.fn();
		render(
			<Stepper
				steps={steps}
				currentStep="step2"
				completedSteps={['step1']}
				onStepClick={onStepClick}
			/>,
		);
		const buttons = screen.getAllByRole('button');
		fireEvent.click(buttons[0]); // Click step1 (completed)
		expect(onStepClick).toHaveBeenCalledWith('step1');
	});

	it('calls onStepClick when the current step is clicked', () => {
		const onStepClick = vi.fn();
		render(
			<Stepper
				steps={steps}
				currentStep="step2"
				completedSteps={['step1']}
				onStepClick={onStepClick}
			/>,
		);
		const buttons = screen.getAllByRole('button');
		fireEvent.click(buttons[1]); // Click step2 (current)
		expect(onStepClick).toHaveBeenCalledWith('step2');
	});

	it('disables future steps', () => {
		const onStepClick = vi.fn();
		render(
			<Stepper
				steps={steps}
				currentStep="step1"
				completedSteps={[]}
				onStepClick={onStepClick}
			/>,
		);
		const buttons = screen.getAllByRole('button');
		expect(buttons[2]).toBeDisabled(); // step3 should be disabled
	});

	it('renders progress nav element', () => {
		render(
			<Stepper steps={steps} currentStep="step1" completedSteps={[]} />,
		);
		expect(screen.getByRole('navigation')).toHaveAttribute(
			'aria-label',
			'Progress',
		);
	});

	it('renders connector lines between steps', () => {
		const { container } = render(
			<Stepper
				steps={steps}
				currentStep="step2"
				completedSteps={['step1']}
			/>,
		);
		const connectors = container.querySelectorAll('.h-0\\.5');
		// Should have connectors between steps (n-1 connectors)
		expect(connectors.length).toBe(steps.length - 1);
	});

	it('applies indigo color to completed connectors', () => {
		const { container } = render(
			<Stepper
				steps={steps}
				currentStep="step2"
				completedSteps={['step1']}
			/>,
		);
		const connectors = container.querySelectorAll('.h-0\\.5');
		expect(connectors[0]).toHaveClass('bg-indigo-600');
	});
});

describe('VerticalStepper', () => {
	it('renders all steps with labels', () => {
		render(
			<VerticalStepper
				steps={steps}
				currentStep="step1"
				completedSteps={[]}
			/>,
		);
		expect(screen.getByText('Step 1')).toBeInTheDocument();
		expect(screen.getByText('Step 2')).toBeInTheDocument();
		expect(screen.getByText('Step 3')).toBeInTheDocument();
	});

	it('renders step descriptions', () => {
		render(
			<VerticalStepper
				steps={steps}
				currentStep="step1"
				completedSteps={[]}
			/>,
		);
		expect(screen.getByText('First step')).toBeInTheDocument();
		expect(screen.getByText('Second step')).toBeInTheDocument();
	});

	it('highlights current step with indigo color', () => {
		render(
			<VerticalStepper
				steps={steps}
				currentStep="step2"
				completedSteps={['step1']}
			/>,
		);
		const step2Label = screen.getByText('Step 2');
		expect(step2Label).toHaveClass('text-indigo-600');
	});

	it('applies completed styling', () => {
		render(
			<VerticalStepper
				steps={steps}
				currentStep="step2"
				completedSteps={['step1']}
			/>,
		);
		const step1Label = screen.getByText('Step 1');
		expect(step1Label).toHaveClass('text-gray-900');
	});

	it('applies pending styling for future steps', () => {
		render(
			<VerticalStepper
				steps={steps}
				currentStep="step1"
				completedSteps={[]}
			/>,
		);
		const step3Label = screen.getByText('Step 3');
		expect(step3Label).toHaveClass('text-gray-500');
	});

	it('calls onStepClick for clickable steps', () => {
		const onStepClick = vi.fn();
		render(
			<VerticalStepper
				steps={steps}
				currentStep="step2"
				completedSteps={['step1']}
				onStepClick={onStepClick}
			/>,
		);
		const buttons = screen.getAllByRole('button');
		fireEvent.click(buttons[0]);
		expect(onStepClick).toHaveBeenCalledWith('step1');
	});

	it('renders navigation with progress label', () => {
		render(
			<VerticalStepper
				steps={steps}
				currentStep="step1"
				completedSteps={[]}
			/>,
		);
		expect(screen.getByRole('navigation')).toHaveAttribute(
			'aria-label',
			'Progress',
		);
	});
});
