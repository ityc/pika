import {Outlet} from 'react-router-dom';
import PublicHeader from '../components/PublicHeader';
import PublicFooter from '../components/PublicFooter';
import {ThemeProvider} from '../contexts/ThemeContext';

const globalStyles = `
    /* 防止滚动时出现白色背景 */
    html, body {
        background-color: #f0f2f5;
    }

    html.dark, body.dark {
        background-color: #05050a;
    }

    /* 整体滚动条 - 亮色模式 */
    ::-webkit-scrollbar {
        width: 8px;
        height: 8px;
    }

    ::-webkit-scrollbar-track {
        background: #e5e7eb;
        border-radius: 4px;
    }

    ::-webkit-scrollbar-thumb {
        background: #9ca3af;
        border-radius: 4px;
        border: 1px solid #d1d5db;
    }

    ::-webkit-scrollbar-thumb:hover {
        background: #6b7280;
    }

    ::-webkit-scrollbar-corner {
        background: #f0f2f5;
    }

    /* Firefox 滚动条 - 亮色模式 */
    * {
        scrollbar-width: thin;
        scrollbar-color: #9ca3af #e5e7eb;
    }

    /* 暗色模式滚动条 */
    .dark ::-webkit-scrollbar-track {
        background: #0a0a0f;
    }

    .dark ::-webkit-scrollbar-thumb {
        background: #1e1e28;
        border: 1px solid #2a2a35;
    }

    .dark ::-webkit-scrollbar-thumb:hover {
        background: #2a2a38;
    }

    .dark ::-webkit-scrollbar-corner {
        background: #05050a;
    }

    /* Firefox 滚动条 - 暗色模式 */
    .dark * {
        scrollbar-color: #1e1e28 #0a0a0f;
    }
`;

const PublicLayout = () => {
    return (
        <ThemeProvider>
            <div className="min-h-screen bg-[#f0f2f5] dark:bg-[#05050a] text-slate-800 dark:text-slate-200 flex flex-col relative overflow-x-hidden transition-colors duration-500">
                <style>{globalStyles}</style>
                {/* 背景网格效果 */}
                <div
                    className="fixed inset-0 pointer-events-none z-0 transition-opacity duration-500"
                    style={{
                        backgroundImage: 'linear-gradient(to_right,#cbd5e180_1px,transparent_1px),linear-gradient(to_bottom,#cbd5e180_1px,transparent_1px)',
                        backgroundSize: '30px 30px',
                    }}
                ></div>
                {/* 暗色模式网格 */}
                <div
                    className="fixed inset-0 pointer-events-none z-0 opacity-0 dark:opacity-100 transition-opacity duration-500"
                    style={{
                        backgroundImage: 'linear-gradient(to_right,#4f4f4f1a_1px,transparent_1px),linear-gradient(to_bottom,#4f4f4f1a_1px,transparent_1px)',
                        backgroundSize: '30px 30px',
                    }}
                ></div>
                {/* 顶部发光效果 */}
                <div
                    className="fixed top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[300px] bg-cyan-600/15 blur-[120px] rounded-full pointer-events-none z-0 opacity-40 dark:opacity-100 transition-opacity duration-500"
                ></div>

                <PublicHeader/>
                <div className="relative z-10 flex flex-col min-h-screen pt-[81px]">
                    <main className="flex-1">
                        <Outlet/>
                    </main>
                    <PublicFooter/>
                </div>
            </div>
        </ThemeProvider>
    );
};

export default PublicLayout;
