import { useContext } from 'react';
import { SiteSettingsContext } from '../contexts/SiteSettingsContext';
import { DEFAULT_SHELTER_TIMEZONE } from '../utils/shelterTime';

/**
 * The shelter's IANA time zone from site settings (AR-5). Unlike
 * useSiteSettings this does not require a provider: without one (e.g. a
 * component rendered alone in a test) it returns UTC, the backend's default.
 */
export function useShelterTimeZone(): string {
  const context = useContext(SiteSettingsContext);
  return context?.settings.shelter_timezone || DEFAULT_SHELTER_TIMEZONE;
}
