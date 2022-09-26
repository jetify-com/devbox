import React from 'react'
import JetpackLogo from '@site/static/img/jetpack_logo_light.svg'
import styles from './styles.module.css';

function HomepageCredits(): JSX.Element {
    return (
    <div className='container credits'>
        <div className='row'>
            <div className={`col ${styles.creditsCol}`}>
                <span><h2>Built by </h2></span>
                <a href='http://www.jetpack.io'>
                <JetpackLogo height={30} width={160} className={styles.jetpackLogo}/>
                </a>
            </div>
        </div>
    </div>
    )
}

export default HomepageCredits
