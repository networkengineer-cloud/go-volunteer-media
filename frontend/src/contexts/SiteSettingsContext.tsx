import React, { createContext, useState, useEffect, useCallback, useMemo } from 'react';
import type { ReactNode } from 'react';
import { settingsApi } from '../api/client';

export interface SiteSettings {
  site_name: string;
  site_short_name: string;
  site_description: string;
  hero_image_url: string;
}

// IMPORTANT: These defaults must match the constants in internal/models/models.go
// (DefaultSiteName, DefaultSiteShortName, DefaultSiteDescription)
// Used as fallbacks when API is unavailable or during initial load
const DEFAULT_SETTINGS: SiteSettings = {
  site_name: 'MyHAWS',
  site_short_name: 'MyHAWS',
  site_description: 'MyHAWS Volunteer Portal - Internal volunteer management system',
  hero_image_url: '',
};

interface SiteSettingsContextType {
  settings: SiteSettings;
  loading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
}

const SiteSettingsContext = createContext<SiteSettingsContextType | undefined>(undefined);

// Export the context so it can be used by the hook in a separate file
export { SiteSettingsContext };

export const SiteSettingsProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [settings, setSettings] = useState<SiteSettings>(DEFAULT_SETTINGS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchSettings = useCallback(async () => {
    try {
      setLoading(true);
      const response = await settingsApi.getAll();
      const data = response.data;
      
      // Merge API data with defaults (in case some settings are missing)
      setSettings({
        site_name: data.site_name || DEFAULT_SETTINGS.site_name,
        site_short_name: data.site_short_name || DEFAULT_SETTINGS.site_short_name,
        site_description: data.site_description || DEFAULT_SETTINGS.site_description,
        hero_image_url: data.hero_image_url || DEFAULT_SETTINGS.hero_image_url,
      });
      setError(null);
    } catch (err) {
      console.error('Failed to fetch site settings:', err);
      setError(err as Error);
      // Keep using default settings on error
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch settings on mount
  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  // Memoize context value to prevent unnecessary re-renders
  const contextValue = useMemo(
    () => ({
      settings,
      loading,
      error,
      refetch: fetchSettings,
    }),
    [settings, loading, error, fetchSettings]
  );

  return (
    <SiteSettingsContext.Provider value={contextValue}>
      {children}
    </SiteSettingsContext.Provider>
  );
};
