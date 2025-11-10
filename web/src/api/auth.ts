import { post } from './request';
import type { LoginRequest, LoginResponse } from '../types';

export const login = (data: LoginRequest) => {
    return post<LoginResponse>('/login', data);
};

export const logout = () => {
    return post('/admin/logout');
};

