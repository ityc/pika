import {createContext, useContext, useEffect, useState, type ReactNode} from 'react';

// 主题类型: light(亮色) | dark(暗色) | auto(自动)
export type Theme = 'light' | 'dark' | 'auto';

// 实际应用的主题: light 或 dark
export type AppliedTheme = 'light' | 'dark';

interface ThemeContextType {
    theme: Theme;
    appliedTheme: AppliedTheme;
    setTheme: (theme: Theme) => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

interface ThemeProviderProps {
    children: ReactNode;
}

export const ThemeProvider = ({children}: ThemeProviderProps) => {
    // 从 localStorage 读取用户偏好,默认为 auto
    const [theme, setThemeState] = useState<Theme>(() => {
        const saved = localStorage.getItem('theme');
        return (saved as Theme) || 'auto';
    });

    // 实际应用的主题
    const [appliedTheme, setAppliedTheme] = useState<AppliedTheme>('dark');

    // 设置主题
    const setTheme = (newTheme: Theme) => {
        setThemeState(newTheme);
        localStorage.setItem('theme', newTheme);
    };

    // 根据主题设置更新实际应用的主题
    useEffect(() => {
        const updateAppliedTheme = () => {
            let applied: AppliedTheme = 'dark';

            if (theme === 'auto') {
                // 自动模式:检测系统主题
                const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                applied = prefersDark ? 'dark' : 'light';
            } else {
                applied = theme;
            }

            setAppliedTheme(applied);

            // 更新 document.documentElement 的 class
            if (applied === 'dark') {
                document.documentElement.classList.add('dark');
            } else {
                document.documentElement.classList.remove('dark');
            }
        };

        updateAppliedTheme();

        // 如果是自动模式,监听系统主题变化
        if (theme === 'auto') {
            const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
            const handler = () => updateAppliedTheme();

            // 使用 addEventListener 替代已废弃的 addListener
            mediaQuery.addEventListener('change', handler);
            return () => mediaQuery.removeEventListener('change', handler);
        }
    }, [theme]);

    return (
        <ThemeContext.Provider value={{theme, appliedTheme, setTheme}}>
            {children}
        </ThemeContext.Provider>
    );
};

// 自定义 Hook 用于访问主题上下文
export const useTheme = () => {
    const context = useContext(ThemeContext);
    if (context === undefined) {
        throw new Error('useTheme must be used within a ThemeProvider');
    }
    return context;
};
