import { DocLink } from 'component/doclink';
import { RefreshButton } from 'component/refreshbutton';
import { Result } from 'component/result';
import { TabController } from 'component/tabcontroller';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { MonospaceContent } from 'f61ui/component/monospacecontent';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import {
	DatabaseBackup,
	DatabaseBackupConfigure,
} from 'generated/stoserver/stoservertypes_commands';
import {
	downloadUbackupStoredBackupUrl,
	getConfig,
	getUbackupStoredBackups,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	CfgUbackupConfig,
	ConfigValue,
	DocRef,
	UbackupStoredBackup,
} from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';
import { metadataBackupUrl } from 'generated/stoserver/stoserverui_uiroutes';

interface MetadataBackupPageProps {
	view: string;
}

interface MetadataBackupPageState {
	backupConfig: Result<ConfigValue>;
	backups: Result<UbackupStoredBackup[]>;
}

export default class MetadataBackupPage extends React.Component<
	MetadataBackupPageProps,
	MetadataBackupPageState
> {
	state: MetadataBackupPageState = {
		backupConfig: new Result<ConfigValue>((_) => {
			this.setState({ backupConfig: _ });
		}),
		backups: new Result<UbackupStoredBackup[]>((_) => {
			this.setState({ backups: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		const content =
			this.props.view === '' ? (
				<Panel
					heading={
						<div>
							Metadata backup list &nbsp;
							<DocLink doc={DocRef.DocsGuideSettingUpBackupMd} />
						</div>
					}>
					{this.renderStoredBackups()}
				</Panel>
			) : (
				this.renderBackupConfig()
			);

		return (
			<SettingsLayout title="Backups" breadcrumbs={[]}>
				<TabController
					tabs={[
						{
							url: metadataBackupUrl({
								view: '',
							}),
							title: 'Metadata backup list',
						},
						{
							url: metadataBackupUrl({
								view: 'config',
							}),
							title: 'Metadata backup configuration',
						},
					]}>
					{content}
				</TabController>
			</SettingsLayout>
		);
	}

	private renderBackupConfig() {
		const [backupConfig, loadingOrError] = this.state.backupConfig.unwrap();

		if (!backupConfig) {
			return loadingOrError;
		}

		const [
			bucket,
			bucketRegion,
			accessKeyId,
			accessKeySecret,
			encryptionPublicKey,
			alertmanagerBaseUrl,
		] = backupConfig.Value ? JSON.parse(backupConfig.Value) : ['', '', '', '', '', ''];

		return (
			<Panel
				heading={
					<div>
						Metadata backup configuration
						<CommandIcon
							command={DatabaseBackupConfigure(
								bucket,
								bucketRegion,
								accessKeyId,
								accessKeySecret,
								encryptionPublicKey,
								alertmanagerBaseUrl,
							)}
						/>
					</div>
				}>
				<table className="table table-striped table-hover">
					<tbody>
						<tr>
							<th>Bucket</th>
							<td>{bucket}</td>
						</tr>
						<tr>
							<th>BucketRegion</th>
							<td>{bucketRegion}</td>
						</tr>
						<tr>
							<th>AccessKeyId</th>
							<td>
								<SecretReveal secret={accessKeyId} />
							</td>
						</tr>
						<tr>
							<th>AccessKeySecret</th>
							<td>
								<SecretReveal secret={accessKeySecret} />
							</td>
						</tr>
						<tr>
							<th>EncryptionPublicKey</th>
							<td>
								<MonospaceContent>{encryptionPublicKey}</MonospaceContent>
							</td>
						</tr>
						<tr>
							<th>AlertmanagerBaseUrl</th>
							<td>
								<SecretReveal secret={alertmanagerBaseUrl} />
							</td>
						</tr>
					</tbody>
				</table>
			</Panel>
		);
	}

	private renderStoredBackups() {
		const [backups, loadingOrError] = this.state.backups.unwrap();

		return (
			<div>
				<table className="table table-striped table-hover">
					<thead>
						<tr>
							<th>Age</th>
							<th>Description</th>
							<th>Size</th>
							<th />
						</tr>
					</thead>
					<tbody>
						{(backups || []).map((backup) => (
							<tr key={backup.ID}>
								<td>
									<Timestamp ts={backup.Timestamp} />
								</td>
								<td>{backup.Description}</td>
								<td>{bytesToHumanReadable(backup.Size)}</td>
								<td>
									<Dropdown>
										<a href={downloadUbackupStoredBackupUrl(backup.ID)}>
											Download
										</a>
									</Dropdown>
								</td>
							</tr>
						))}
					</tbody>
					<tfoot>
						<tr>
							<td colSpan={99}>
								<div>{loadingOrError}</div>
								<div>
									<RefreshButton
										refresh={() => {
											this.loadBackups();
										}}
									/>
								</div>
								<CommandButton command={DatabaseBackup()} />
							</td>
						</tr>
					</tfoot>
				</table>
			</div>
		);
	}

	private fetchData() {
		this.state.backupConfig.load(() => getConfig(CfgUbackupConfig));

		this.loadBackups();
	}

	private loadBackups() {
		this.state.backups.load(() => getUbackupStoredBackups());
	}
}
