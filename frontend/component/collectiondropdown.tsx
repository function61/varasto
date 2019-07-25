import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import {
	CollectionChangeDescription,
	CollectionChangeSensitivity,
	CollectionDelete,
	CollectionFuseMount,
	CollectionMove,
	CollectionPullMetadata,
	CollectionRename,
} from 'generated/stoserver_commands';
import { CollectionSubset } from 'generated/stoserver_types';
import * as React from 'react';

export function collectionDropdown(coll: CollectionSubset) {
	return (
		<Dropdown>
			<CommandLink command={CollectionRename(coll.Id, coll.Name)} />
			<CommandLink command={CollectionChangeDescription(coll.Id, coll.Description)} />
			<CommandLink command={CollectionMove(coll.Id)} />
			<CommandLink command={CollectionChangeSensitivity(coll.Id, coll.Sensitivity)} />
			<CommandLink command={CollectionFuseMount(coll.Id)} />
			<CommandLink command={CollectionPullMetadata(coll.Id)} />
			<CommandLink command={CollectionDelete(coll.Id)} />
		</Dropdown>
	);
}
