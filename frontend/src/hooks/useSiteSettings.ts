import { useState, useEffect } from 'react';
import { settingsApi } from '../api/client';

export interface SiteSettings {
  site_name: string;
  site_short_name: string;
  site_description: string;
  hero_image_url: string;
}

const DEFAULT_SETTINGS: SiteSettings = {
  site_name: 'MyHAWS',
  site_short_name: 'MyHAWS',
  site_description: 'MyHAWS Volunteer Portal - Internal volunteer management system',
  hero_image_url: '',
};

/**
 * Custom hook to fetch and cache site settings from the API.
 * Provides sensible defaults while loading and on error.
 */
export function useSiteSettings() {
  const [settings, setSettings] = useState<SiteSettings>(DEFAULT_SETTINGS);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const fetchSettings = async () => {
      try {
        const response = await settingsApi.getAll();
        const data = response.data;
        
        // Merge API data with defaults (in case some settings are missing)
        setSettings({
          site_name: data.site_name || DEFAULT_SETTINGS.site_name,
          site_short_name: data.site_short_name || DEFAULT_SETTINGS.site_short_name,
          site_description: data.site_description || DEFAULT_SETTINGS.site_description,
          hero_image_url: data.hero_image_url || DEFAULT_SETTINGS.hero_image_url,
        });
      } catch (err) {
        console.error('Failed to fetch site settings:', err);
        setError(err as Error);
        // Keep using default settings on error
      } finally {
        setLoading(false);
      }
    };

    fetchSettings();
  }, []);

  return { settings, loading, error };
}
