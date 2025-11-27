// ElectricSQL client configuration
// This will be fully implemented once we test the basic API flow

const ELECTRIC_URL = import.meta.env.VITE_ELECTRIC_URL || 'http://localhost:3000';

export interface ElectricConfig {
	url: string;
}

export const electricConfig: ElectricConfig = {
	url: ELECTRIC_URL
};

// Placeholder for Electric client
// Will be implemented after Phase 1 is complete
export const electric = {
	config: electricConfig,
	// Future: shape subscriptions will go here
};

export default electric;
