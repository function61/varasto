import { SearchBox } from 'component/autocomplete';
import { CustomFieldInputFactory } from 'f61ui/commandtypes';
import {
	searchMetadataImdbMovieId,
	searchIgdb,
	searchReplicationPolicies,
	searchVolumes,
} from 'generated/stoserver/stoservertypes_endpoints';
import * as React from 'react';

export const replicationPolicyAutocomplete: CustomFieldInputFactory<string> = (
	field,
	_,
	update,
	autoFocus,
) => {
	return (
		<SearchBox
			allowEmptySearch={false}
			autoFocus={autoFocus}
			placeholder={field.Placeholder}
			searchTerm={field.DefaultValueString}
			dataSource={searchReplicationPolicies}
			onSelect={(item) => {
				update(item.key);
			}}
			itemToAutocompleteItem={(item) => {
				return { label: item.Name, key: item.Id };
			}}
		/>
	);
};

export const volumeAutocomplete: CustomFieldInputFactory<number> = (
	field,
	_,
	update,
	autoFocus,
) => {
	return (
		<SearchBox
			allowEmptySearch={false}
			autoFocus={autoFocus}
			placeholder={field.Placeholder}
			dataSource={searchVolumes}
			onSelect={(item) => {
				if (item.key) {
					update(+item.key);
				} else {
					update(undefined);
				}
			}}
			itemToAutocompleteItem={(vol) => {
				return { label: vol.Label, key: vol.Id.toString() };
			}}
		/>
	);
};

export const tmdbAutocomplete: CustomFieldInputFactory<string> = (field, _, update, autoFocus) => {
	return (
		<SearchBox
			allowEmptySearch={false}
			autoFocus={autoFocus}
			placeholder={field.Placeholder}
			dataSource={searchMetadataImdbMovieId}
			onSelect={(item) => {
				update(item.key);
			}}
			itemToAutocompleteItem={(item) => {
				const releaseYear = item.ReleaseYear;
				const label = releaseYear ? `${item.Title} (${releaseYear})` : item.Title;

				return { label, key: item.Id };
			}}
		/>
	);
};

export const igdbAutocomplete: CustomFieldInputFactory<string> = (field, _, update, autoFocus) => {
	return (
		<SearchBox
			allowEmptySearch={false}
			autoFocus={autoFocus}
			placeholder={field.Placeholder}
			dataSource={searchIgdb}
			onSelect={(item) => {
				update(item.key);
			}}
			itemToAutocompleteItem={(item) => {
				const releaseYear = item.ReleaseYear;
				const label = releaseYear ? `${item.Title} (${releaseYear})` : item.Title;

				return { label, key: item.Id };
			}}
		/>
	);
};
