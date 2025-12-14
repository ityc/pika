import {SnapshotGrid, type SnapshotCardData} from './SnapshotGrid';

interface SnapshotSectionProps {
    cards: SnapshotCardData[];
}

export const SnapshotSection = ({cards}: SnapshotSectionProps) => {
    if (cards.length === 0) {
        return null;
    }
    return (
        <div className="space-y-4">
            <div>
                <h3 className="text-xs font-bold font-mono uppercase tracking-widest text-cyan-500">资源快照</h3>
            </div>
            <SnapshotGrid cards={cards}/>
        </div>
    );
};
