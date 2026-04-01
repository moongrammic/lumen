import axios from "axios";
import { useAuthStore } from "@/store/useAuthStore";

const baseURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api";

export const api = axios.create({
  baseURL,
  withCredentials: true,
});

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      await useAuthStore.getState().logout();
      if (typeof window !== "undefined") {
        window.location.assign("/login");
      }
    }
    return Promise.reject(error);
  },
);
