import {type ClassValue, clsx} from 'clsx';
import {twMerge} from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

interface ErrorWithResponse {
    response?: {
        data?: {
            message?: string;
        };
    };
}

export function getErrorMessage(error: unknown, fallback: string) {
    if (typeof error === 'object' && error !== null) {
        const {response} = error as ErrorWithResponse;
        if (response?.data?.message) {
            return response.data.message;
        }
    }

    if (error instanceof Error && error.message) {
        return error.message;
    }

    return fallback;
}
