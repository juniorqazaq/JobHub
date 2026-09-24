import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { repositories } from "../api/repositories";
import { useAuth } from "../app/authContext";
import { isCityId, type CityId } from "./cities";

const key = "jobhub.searchCity";
export const searchCityChanged = "jobhub:search-city-changed";

export function getSearchCityPreference(): CityId | undefined {
  if (typeof window === "undefined") return undefined;
  const value = window.localStorage.getItem(key);
  return isCityId(value) ? value : undefined;
}

export function setSearchCityPreference(value?: CityId) {
  if (value) window.localStorage.setItem(key, value);
  else window.localStorage.removeItem(key);
  window.dispatchEvent(new CustomEvent(searchCityChanged, { detail: value }));
}

export function usePreferredSearchCity() {
  const { session } = useAuth();
  const [localCity, setLocalCity] = useState(getSearchCityPreference);
  const profile = useQuery({
    queryKey: ["candidate-profile"],
    queryFn: () => repositories.candidate.getProfile(),
    enabled: session?.user.role === "job_seeker",
  });

  useEffect(() => {
    const sync = () => setLocalCity(getSearchCityPreference());
    window.addEventListener(searchCityChanged, sync);
    return () => window.removeEventListener(searchCityChanged, sync);
  }, []);

  const profileCity = profile.data?.preferredCityIds.find(isCityId) ?? profile.data?.cityId;
  return localCity ?? (isCityId(profileCity) ? profileCity : undefined);
}
