import { useContext } from 'react';
import { SiteSettingsContext } from '../contexts/SiteSettingsContext';

/**
 * Custom hook to access site settings from the SiteSettingsContext.
 * Settings are fetched once at the app level and shared across all consumers.
 * Provides a refetch() function for manual refresh (e.g., after admin updates).
 */
export function useSiteSettings() {
  const context = useContext(SiteSettingsContext);
  
  if (context === undefined) {
    throw new Error('useSiteSettings must be used within a SiteSettingsProvider');
  }
  
  return context;
}
