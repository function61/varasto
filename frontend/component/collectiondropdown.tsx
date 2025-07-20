import { igdbAutocomplete, tmdbMovieAutocomplete } from 'component/autocompletes';
import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import {
	CollectionChangeDescription,
	CollectionChangeSensitivity,
	CollectionDelete,
	CollectionMove,
	CollectionPullIgdbMetadata,
	CollectionPullTmdbMetadata,
	CollectionRename,
	CollectionTag,
	CollectionTriggerMediaScan,
} from 'generated/stoserver/stoservertypes_commands';
import { CollectionSubset } from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

export function collectionDropdown(coll: CollectionSubset) {
	return (
		<Dropdown>
			<CommandLink command={CollectionRename(coll.Id, coll.Name)} />
			<CommandLink command={CollectionChangeDescription(coll.Id, coll.Description)} />
			<CommandLink command={CollectionMove([coll.Id], { disambiguation: coll.Name })} />
			<CommandLink command={CollectionChangeSensitivity(coll.Id, coll.Sensitivity)} />
			<CommandLink
				command={CollectionPullTmdbMetadata(coll.Id, { ForeignKey: tmdbMovieAutocomplete })}
			/>
			<CommandLink
				command={CollectionPullIgdbMetadata(coll.Id, { ForeignKey: igdbAutocomplete })}
			/>
			<CommandLink command={CollectionTriggerMediaScan(coll.Id)} />
			<CommandLink command={CollectionTag(coll.Id, { disambiguation: coll.Name })} />
			<CommandLink command={CollectionDelete(coll.Id, { disambiguation: coll.Name })} />
		</Dropdown>
	);
}
