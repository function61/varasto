import { DocLink } from 'component/doclink';
import { WarningAlert } from 'f61ui/component/alerts';
import { DefaultLabel, Glyphicon, Panel } from 'f61ui/component/bootstrap';
import { Info } from 'f61ui/component/info';
import { unrecognizedValue } from 'f61ui/utils';
import {
	CollectionCreate,
	KekGenerateOrImport,
	ReplicationpolicyChangeDesiredVolumes,
	VolumeCreate,
	VolumeMountGoogleDrive,
	VolumeMountLocal,
	VolumeMountS3,
} from 'generated/stoserver/stoservertypes_commands';
import { DocRef, RootFolderId } from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import {
	browseUrl,
	downloadClientAppUrl,
	gettingStartedUrl,
	replicationPoliciesUrl,
	usersUrl,
	volumesUrl,
	mountsUrl,
} from 'generated/stoserver/stoserverui_uiroutes';

interface SmallWellProps {
	children: React.ReactNode;
}

class SmallWell extends React.Component<SmallWellProps, {}> {
	render() {
		return <span className="well well-sm">{this.props.children}</span>;
	}
}

type section =
	| 'welcome'
	| 'createUser'
	| 'setUpEncryption'
	| 'yourFirstVolume'
	| 'mountFirstVolume'
	| 'defaultReplicationPolicy'
	| 'createFirstCollection'
	| 'addingFilesToCollection'
	| 'cloningCollection'
	| 'done';

interface GettingStartedPageProps {
	view: string;
}

interface GettingStartedPageState {
	careAboutKek?: string;
	mountType?: string;
}

export default class GettingStartedPage extends React.Component<
	GettingStartedPageProps,
	GettingStartedPageState
> {
	state: GettingStartedPageState = {};

	render() {
		let preceded = true;

		const panel = (
			category: string | null,
			title: string,
			viewId: section,
			fn: (currSection: section) => React.ReactNode,
		): React.ReactNode => {
			const isCurr = viewId === this.props.view;

			if (isCurr) {
				preceded = false;
			}

			const heading = (
				<span>
					{preceded && <Glyphicon icon="ok" />}
					&nbsp;
					{category && <DefaultLabel>{category}</DefaultLabel>}
					&nbsp;
					{title}
				</span>
			);

			return <Panel heading={heading}>{isCurr && fn.call(this, viewId)}</Panel>;
		};

		return (
			<AppDefaultLayout title="Getting started" breadcrumbs={[]}>
				{panel(null, 'Welcome to Varasto!', 'welcome', this.welcome)}
				{panel(null, 'Create user', 'createUser', this.createUser)}
				{panel(null, 'Set up encryption', 'setUpEncryption', this.setUpEncryption)}
				{panel(null, 'Your first volume', 'yourFirstVolume', this.yourFirstVolume)}
				{panel(null, 'Mount first volume', 'mountFirstVolume', this.mountFirstVolume)}
				{panel(
					null,
					'Default replication policy',
					'defaultReplicationPolicy',
					this.defaultReplicationPolicy,
				)}
				{panel(
					'Tutorial',
					'Create first collection',
					'createFirstCollection',
					this.createFirstCollection,
				)}
				{panel(
					'Tutorial',
					'Adding files to collection',
					'addingFilesToCollection',
					this.addingFilesToCollection,
				)}
				{panel(
					'Tutorial',
					'Cloning a collection to your computer',
					'cloningCollection',
					this.cloningCollection,
				)}
				{panel(null, 'Done! Links to further documentation', 'done', this.done)}
			</AppDefaultLayout>
		);
	}

	private welcome(currSection: section): React.ReactNode {
		return (
			<div>
				<p>
					We have worked really hard to make an easy-to-use, self-guiding and an enjoyable
					user experience.
				</p>
				<p>This will be a guided tour of:</p>
				<ul>
					<li>setting up Varasto and</li>
					<li>(optionally) learning to use it with tutorials</li>
				</ul>
				<h3>Pro-tip: info tooltips</h3>
				<p>
					Every time you see icon like this:{' '}
					<Info text="Here will be helpful info text." /> you should hover over it,
					because we've written useful tips to help understand this system.
				</p>

				<h3>Pro-tip: documentation links</h3>

				<p>
					An icon like this takes you to documentation:{' '}
					<DocLink doc={DocRef.DocsUsingNetworkFoldersIndexMd} />. We have written the
					documentation with great love, to try to make it short enough to read but still
					give you a good picture of how things work!
				</p>

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private createUser(currSection: section): React.ReactNode {
		return (
			<div>
				<p>TODO: Currently Varasto isn't a multi-user system.</p>

				<p>This will get fixed shortly.</p>

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private setUpEncryption(currSection: section): React.ReactNode {
		const kekCreateUrl = usersUrl();

		const iDontCareAboutKek = 'I don´t know or care what this key is';

		const kekCareQuestionChange = (value: string) => {
			this.setState({ careAboutKek: value });
		};

		return (
			<div>
				<p>We need to configure a key encryption key ("KEK").</p>

				<p>
					Please read at least the <SmallWell>Summary</SmallWell> section of docs:{' '}
					<DocLink doc={DocRef.DocsSecurityEncryptionIndexMd} />
				</p>

				<p>
					Do you know what a key encryption key is, and do you have an existing one you
					want to use with Varasto?
				</p>

				{mkRadio(
					'kekCareQuestion',
					'no',
					kekCareQuestionChange,
					'I don´t know what a KEK is or I don´t want to import one - let Varasto generate it for me!',
				)}
				{mkRadio(
					'kekCareQuestion',
					'yes',
					kekCareQuestionChange,
					'I have an existing KEK I want to import into Varasto!',
				)}

				<hr />

				{this.state.careAboutKek === 'no' && (
					<div>
						<p>Here's how to generate a new KEK in Varasto:</p>

						<ul>
							<li>
								Go to{' '}
								<a href={kekCreateUrl} target="_blank">
									manage KEKs
								</a>
							</li>
							<li>
								Click{' '}
								<SmallWell>
									Key encryption keys &raquo; {KekGenerateOrImport().title}
								</SmallWell>
							</li>
							<li>
								Leave <SmallWell>Import existing</SmallWell> blank
							</li>
						</ul>
					</div>
				)}

				{this.state.careAboutKek === 'yes' && (
					<div>
						<p>
							If you have a KEK you want to use that is managed outside of Varasto,
							and you'd like to import only the public key to Varasto, that's planned
							but not yet implemented. :(
						</p>

						<p>
							Until{' '}
							<a
								href="https://github.com/function61/varasto/issues/133"
								target="_blank">
								this
							</a>{' '}
							gets implemented, Varasto still needs access to the private key. Your
							options are:
						</p>

						<ul>
							<li>
								Let Varasto create a new KEK by following the section{' '}
								<SmallWell>{iDontCareAboutKek}</SmallWell>. Once Varasto gets
								"public key only" support, you can easily migrate all your previous
								data to another KEK public key.
							</li>
							<li>Import your existing private KEK key to Varasto</li>
						</ul>
					</div>
				)}

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private yourFirstVolume(currSection: section): React.ReactNode {
		return (
			<div>
				<p>A volume is a physical storage location for your data. That can be a:</p>
				<ul>
					<li>a directory in your existing partition</li>
					<li>a dedicated partition</li>
					<li>a cloud service</li>
				</ul>
				<p>
					You can store data in multiple volumes for redundancy. If a disk breaks, you
					still have the same data in another volume so you won't lose data.
				</p>
				<p>
					But you don't have to worry about redundancy choices right now - you can add
					volumes later and it's easy to change replication settings even for existing
					data to be spread to two or more volumes later.
				</p>
				<p>
					Note: you don't need to decide where data is stored when you create a volume -
					you'll make that decision when you mount the volume (that's the next page).
				</p>

				<p>
					A default volume with a 1 GB quota has been created for you. You can change its
					name and quota now or later.
				</p>

				<p>
					Go to{' '}
					<a href={volumesUrl()} target="_blank">
						volumes
					</a>{' '}
					to see your first volume. If you want to create additional volumes, use{' '}
					<SmallWell>Volumes &raquo; {VolumeCreate().title}</SmallWell>.
				</p>

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private mountFirstVolume(currSection: section): React.ReactNode {
		const mountTypeChange = (value: string) => {
			this.setState({ mountType: value });
		};

		return (
			<div>
				<p>It's time to mount the volume(s) that you created.</p>

				<p>
					When you mount a volume for the first time, you'll decide where the data is
					actually stored.
				</p>

				<p>I would like to mount a:</p>

				{mkRadio('mountType', 'localDisk', mountTypeChange, 'A local disk or a directory')}

				{mkRadio('mountType', 's3', mountTypeChange, 'AWS S3')}

				{mkRadio('mountType', 'googleDrive', mountTypeChange, 'Google Drive (& G Suite)')}

				<hr />

				<p>
					Go to{' '}
					<a href={mountsUrl()} target="_blank">
						Mounts
					</a>
					. From there choose:
				</p>

				{this.state.mountType === 'localDisk' && (
					<div>
						<SmallWell>Volume &raquo; {VolumeMountLocal(0).title}</SmallWell>{' '}
						<WarningAlert>
							Once you've selected it, be sure to read <b>important documentation</b>{' '}
							behind this icon:&nbsp;
							<DocLink doc={DocRef.DocsStorageLocalFsIndexMd} />
						</WarningAlert>
					</div>
				)}

				{this.state.mountType === 's3' && (
					<div>
						<SmallWell>Volume &raquo; {VolumeMountS3(0).title}</SmallWell>{' '}
						<DocLink doc={DocRef.DocsStorageS3IndexMd} />
					</div>
				)}

				{this.state.mountType === 'googleDrive' && (
					<div>
						<SmallWell>Volume &raquo; {VolumeMountGoogleDrive(0).title}</SmallWell>{' '}
						<DocLink doc={DocRef.DocsStorageGoogledriveIndexMd} />
					</div>
				)}

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private defaultReplicationPolicy(currSection: section): React.ReactNode {
		return (
			<div>
				<p>
					A replication policy defines into which volumes a collection's files are stored
					in.
				</p>
				<p>You can have separate policies for data of varying types of importance:</p>
				<ul>
					<li>For data you're OK with losing, you can use just one volume.</li>
					<li>For more important data, you can use two or more volumes.</li>
					<li>
						You should also consider geographic location so that your important data is
						safe even if a fire destroys the primary location of your data.
					</li>
				</ul>

				<p>
					Read more: <DocLink doc={DocRef.DocsUsingReplicationPoliciesIndexMd} />{' '}
					(includes a picture)
				</p>

				<p>
					<b>A default policy was created for you</b>, which specifies that data is stored
					on your default volume. After you add more volumes, you can change the
					replication policy (even retroactively) to increase your data redundancy.
				</p>

				<p>
					Go to{' '}
					<a href={replicationPoliciesUrl()} target="_blank">
						replication policies
					</a>{' '}
					to see your default replication policy.
				</p>

				<p>
					If you wish to change the policy, use{' '}
					<SmallWell>{ReplicationpolicyChangeDesiredVolumes('0').title}</SmallWell> to
					specify which volumes your data should be written to.
				</p>

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private createFirstCollection(currSection: section): React.ReactNode {
		return (
			<div>
				<p>
					Go to{' '}
					<a href={browseUrl({ dir: RootFolderId, view: '' })} target="_blank">
						browse
					</a>{' '}
					and click <SmallWell>{CollectionCreate('').title}</SmallWell>.
				</p>

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private addingFilesToCollection(currSection: section): React.ReactNode {
		return (
			<div>
				<p>Open the collection that you just created.</p>
				<p>
					Look for <SmallWell>Upload &raquo; Choose files</SmallWell>. You can drag-n-drop
					files over that or use the <SmallWell>Choose files</SmallWell> button.
				</p>

				<p>This is the most basic way of managing files. You can also manage files:</p>

				<ul>
					<li>From CLI</li>
					<li>Over network folders</li>
					<li>Backup client</li>
				</ul>

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private cloningCollection(currSection: section): React.ReactNode {
		return (
			<div>
				<WarningAlert>
					NOTE: Remember to install client app first:{' '}
					<a href={downloadClientAppUrl()} target="_blank">
						links &amp; instructions
					</a>
				</WarningAlert>

				<p>
					The example collection that you created and added files to. Click{' '}
					<SmallWell>Details &raquo; Clone command &raquo; Clipboard icon</SmallWell> to
					copy the command to clipboard.
				</p>

				<p>
					The command now in clipboard looks like <SmallWell>sto clone ...</SmallWell>
				</p>

				<p>
					Run this in your computer's command prompt / terminal. It'll clone the
					collection and its content to your computer so you can make modifications
					locally.
				</p>

				<p>
					To demonstrate making changes, add a file to the cloned collection. Then run{' '}
					<SmallWell>sto push</SmallWell> to send the changes back to Varasto.
				</p>

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private done(currSection: section): React.ReactNode {
		return (
			<div>
				<p>Done! 🎉 You're awesome for reaching this far! 💪</p>

				<p>
					For advanced help, check out the main documentation!{' '}
					<DocLink doc={DocRef.READMEMd} />
				</p>

				<p>
					Consider subscribing to{' '}
					<a href="https://buttondown.email/varasto" target="_blank">
						Varasto mailing list
					</a>{' '}
					by email (or RSS) to stay on top of news &amp; updates!
				</p>

				{this.phaseNavBar(currSection)}
			</div>
		);
	}

	private phaseNavBar(currSection: section): React.ReactNode {
		const url = this.nextUrl(currSection);

		const hasPrevious = currSection !== 'welcome';

		return (
			<div>
				<hr />
				{hasPrevious && (
					<a href="javascript:history.back()" className="btn btn-default">
						Previous
					</a>
				)}
				&nbsp;
				{url && (
					<a href={url} className="btn btn-primary">
						Next
					</a>
				)}
			</div>
		);
	}

	private nextUrl(currSection: section): string | null {
		const nextSection = this.nextSection(currSection);

		if (!nextSection) {
			return null;
		}

		return gettingStartedUrl({ section: nextSection });
	}

	private nextSection(currSection: section): section | null {
		switch (currSection) {
			case 'welcome':
				return 'createUser';
			case 'createUser':
				return 'setUpEncryption';
			case 'setUpEncryption':
				return 'yourFirstVolume';
			case 'yourFirstVolume':
				return 'mountFirstVolume';
			case 'mountFirstVolume':
				return 'defaultReplicationPolicy';
			case 'defaultReplicationPolicy':
				return 'createFirstCollection';
			case 'createFirstCollection':
				return 'addingFilesToCollection';
			case 'addingFilesToCollection':
				return 'cloningCollection';
			case 'cloningCollection':
				return 'done';
			case 'done':
				return null;
			default:
				throw unrecognizedValue(currSection);
		}
	}
}

function mkRadio(
	name: string,
	value: string,
	selected: (value: string) => void,
	label: React.ReactNode,
): React.ReactNode {
	return (
		<div>
			<label>
				<input
					type="radio"
					name={name}
					value={value}
					onChange={(e) => {
						selected(e.target.value);
					}}
				/>
				&nbsp; {label}
			</label>
		</div>
	);
}
