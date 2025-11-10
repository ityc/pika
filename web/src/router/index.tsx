import { createBrowserRouter, Navigate } from 'react-router-dom';
import { lazy, Suspense, type ComponentType, type LazyExoticComponent } from 'react';
import PrivateRoute from '../components/PrivateRoute';

const LoginPage = lazy(() => import('../pages/Login'));
const AdminLayout = lazy(() => import('../pages/AdminLayout'));
const AgentListPage = lazy(() => import('../pages/Agents/AgentList'));
const AgentDetailPage = lazy(() => import('../pages/Agents/AgentDetail'));
const AgentInstallPage = lazy(() => import('../pages/Agents/AgentInstall'));
const AuditResultPage = lazy(() => import('../pages/Agents/AuditResult'));
const UserListPage = lazy(() => import('../pages/Users/UserList'));
const ApiKeyListPage = lazy(() => import('../pages/ApiKeys/ApiKeyList'));
const SettingsPage = lazy(() => import('../pages/Settings'));
const ServerListPage = lazy(() => import('../pages/Public/ServerList'));
const ServerDetailPage = lazy(() => import('../pages/Public/ServerDetail'));

const LoadingFallback = () => (
    <div className="flex min-h-[200px] w-full items-center justify-center text-gray-500">
        页面加载中...
    </div>
);

const lazyLoad = (Component: LazyExoticComponent<ComponentType<any>>) => (
    <Suspense fallback={<LoadingFallback />}>
        <Component />
    </Suspense>
);

const router = createBrowserRouter([
    // 登录页面
    {
        path: '/login',
        element: lazyLoad(LoginPage),
    },
    // 公开页面 - 不需要登录
    {
        path: '/',
        element: lazyLoad(ServerListPage),
    },
    {
        path: '/servers/:id',
        element: lazyLoad(ServerDetailPage),
    },
    // 管理员页面 - 需要登录
    {
        path: '/admin',
        element: (
            <PrivateRoute>
                {lazyLoad(AdminLayout)}
            </PrivateRoute>
        ),
        children: [
            {
                index: true,
                element: <Navigate to="/admin/agents" replace />,
            },
            {
                path: 'agents',
                element: lazyLoad(AgentListPage),
            },
            {
                path: 'agents/:id',
                element: lazyLoad(AgentDetailPage),
            },
            {
                path: 'agents/:id/audit',
                element: lazyLoad(AuditResultPage),
            },
            {
                path: 'agents-install',
                element: lazyLoad(AgentInstallPage),
            },
            {
                path: 'api-keys',
                element: lazyLoad(ApiKeyListPage),
            },
            {
                path: 'users',
                element: lazyLoad(UserListPage),
            },
            {
                path: 'settings',
                element: lazyLoad(SettingsPage),
            },
        ],
    },
]);

export default router;
