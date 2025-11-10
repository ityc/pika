import { get, post, put, del } from './request';
import type { User, CreateUserRequest, UpdateUserRequest, ChangePasswordRequest } from '../types';

export interface ListUsersResponse {
    items: User[];
    total: number;
}

export const listUsers = (page: number = 1, pageSize: number = 10, username?: string) => {
    const params = new URLSearchParams();
    params.append('page', page.toString());
    params.append('pageSize', pageSize.toString());
    if (username) {
        params.append('username', username);
    }
    return get<ListUsersResponse>(`/admin/users?${params.toString()}`);
};

export const getUser = (id: string) => {
    return get<User>(`/admin/users/${id}`);
};

export const createUser = (data: CreateUserRequest) => {
    return post<User>('/admin/users', data);
};

export const updateUser = (id: string, data: UpdateUserRequest) => {
    return put<User>(`/admin/users/${id}`, data);
};

export const deleteUser = (id: string) => {
    return del(`/admin/users/${id}`);
};

export const changePassword = (id: string, data: ChangePasswordRequest) => {
    return post(`/admin/users/${id}/password`, data);
};

export const resetPassword = (id: string, newPassword: string) => {
    return post(`/admin/users/${id}/reset-password`, { newPassword });
};

export const updateUserStatus = (id: string, status: number) => {
    return post(`/admin/users/${id}/status`, { status });
};

