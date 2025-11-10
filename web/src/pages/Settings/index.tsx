import {Tabs} from 'antd';
import {Bell} from 'lucide-react';
import AlertSettings from './AlertSettings';
import {PageHeader} from "../../components";

const Settings = () => {
    const items = [
        {
            key: 'alert',
            label: (
                <span className="flex items-center gap-2">
                    <Bell size={16}/>
                    告警配置
                </span>
            ),
            children: <AlertSettings/>,
        },
    ];

    return (
        <div className={'space-y-6'}>
            <PageHeader
                title="系统设置"
                description="CONFIGURATION"
            />
            <Tabs defaultActiveKey="alert"
                  tabPosition={'left'}
                  items={items}
            />
        </div>
    );
};

export default Settings;
